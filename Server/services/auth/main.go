package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	// Application layer
	"github.com/yourusername/connect/auth-service/application/commands/login"
	"github.com/yourusername/connect/auth-service/application/commands/logout"
	"github.com/yourusername/connect/auth-service/application/commands/refresh_token"
	"github.com/yourusername/connect/auth-service/application/commands/register"
	"github.com/yourusername/connect/auth-service/application/queries/validate_token"

	// Infrastructure layer
	grpcServer "github.com/yourusername/connect/auth-service/infrastructure/grpc"
	"github.com/yourusername/connect/auth-service/infrastructure/messaging"
	"github.com/yourusername/connect/auth-service/infrastructure/persistence"
	"github.com/yourusername/connect/auth-service/infrastructure/security"
	"github.com/yourusername/connect/auth-service/infrastructure/validation"

	// Config
	"github.com/yourusername/connect/auth-service/config"
	"github.com/yourusername/connect/auth-service/infrastructure/ratelimit"

	// Proto
	pb "github.com/yourusername/connect/proto/auth"

	// Shared packages
	"github.com/yourusername/connect/pkg/database"
	"github.com/yourusername/connect/pkg/grpctls"
	"github.com/yourusername/connect/pkg/health"
	"github.com/yourusername/connect/pkg/logger"
	"github.com/yourusername/connect/pkg/metrics"
	"github.com/yourusername/connect/pkg/requestid"
	"github.com/yourusername/connect/pkg/tracing"
)

func main() {
	log := logger.New()
	defer log.Sync()

	cfg := config.LoadConfig()

	// Initialize OpenTelemetry tracing
	shutdownTracer, err := tracing.Init(context.Background(), tracing.Config{
		ServiceName: "auth-service",
		Endpoint:    cfg.OTELEndpoint,
	}, log)
	if err != nil {
		log.Fatal("failed to init tracing", zap.Error(err))
	}
	defer shutdownTracer(context.Background())

	// Initialize PostgreSQL
	db, err := database.Connect(database.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		Name:     cfg.DBName,
		SSL:      cfg.DBSSL,
	}, log)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("error closing database", zap.Error(err))
		}
	}()

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password:     cfg.RedisPassword,
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})
	defer redisClient.Close()

	// Verify Redis connectivity
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatal("failed to ping Redis", zap.Error(err))
	}
	log.Info("connected to Redis", zap.String("addr", cfg.RedisHost+":"+cfg.RedisPort))

	// Create adapters (Infrastructure layer)
	userRepo := persistence.NewPostgresUserRepository(db)
	passwordHasher := security.NewBcryptHasher()

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	accessExpiry, err := time.ParseDuration(cfg.JWTAccessExpiry)
	if err != nil {
		log.Fatal("invalid JWT_ACCESS_EXPIRY", zap.String("value", cfg.JWTAccessExpiry), zap.Error(err))
	}
	refreshExpiry, err := time.ParseDuration(cfg.JWTRefreshExpiry)
	if err != nil {
		log.Fatal("invalid JWT_REFRESH_EXPIRY", zap.String("value", cfg.JWTRefreshExpiry), zap.Error(err))
	}
	tokenGenerator := security.NewJWTTokenGenerator(cfg.JWTSecret, accessExpiry, refreshExpiry)
	tokenValidator := security.NewJWTTokenValidator(cfg.JWTSecret)
	tokenBlacklist := security.NewRedisTokenBlacklist(redisClient)
	rateLimiter := security.NewRedisRateLimiter(redisClient, cfg.LoginMaxAttempts, cfg.LoginWindow)
	validator := validation.NewInputValidator()

	// Initialize RabbitMQ event publisher
	eventPublisher, err := messaging.NewRabbitMQEventPublisher(cfg.RabbitMQURL, "auth_events")
	if err != nil {
		log.Warn("failed to initialize RabbitMQ publisher (user profile sync disabled)", zap.Error(err))
	} else {
		defer eventPublisher.Close()
	}

	// Create application handlers (Application layer)
	registerHandler := register.NewHandler(userRepo, passwordHasher, tokenGenerator, validator, eventPublisher)
	loginHandler := login.NewHandler(userRepo, passwordHasher, tokenGenerator, rateLimiter, log)
	validateTokenHandler := validate_token.NewHandler(tokenValidator, tokenBlacklist)
	refreshTokenHandler := refresh_token.NewHandler(userRepo, tokenValidator, tokenGenerator, tokenBlacklist)
	logoutHandler := logout.NewHandler(tokenValidator, tokenBlacklist)

	// Create gRPC service (Presentation layer)
	authGRPCService := grpcServer.NewAuthGRPCService(
		registerHandler,
		loginHandler,
		validateTokenHandler,
		refreshTokenHandler,
		logoutHandler,
	)

	// Create rate limiter (60 requests per minute)
	rl := ratelimit.NewRateLimiter(cfg.GRPCRateLimit, cfg.GRPCRateWindow)

	// Prometheus gRPC metrics
	grpcMetrics := metrics.NewGRPCMetrics("auth_service")

	// Create gRPC server with metrics and rate limit interceptors
	serverOpts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(4 * 1024 * 1024),
		grpc.StatsHandler(tracing.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			requestid.UnaryServerInterceptor(),
			metrics.UnaryServerInterceptor(grpcMetrics),
			ratelimit.UnaryRateLimitInterceptor(rl),
		),
	}
	if tlsCreds, err := grpctls.ServerOptions(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil {
		log.Fatal("failed to load TLS credentials", zap.Error(err))
	} else if tlsCreds != nil {
		serverOpts = append(serverOpts, tlsCreds)
		log.Info("gRPC TLS enabled")
	}
	server := grpc.NewServer(serverOpts...)

	pb.RegisterAuthServiceServer(server, authGRPCService)
	reflection.Register(server)

	// Register gRPC health check
	healthSrv := health.Register(server)

	// Start Prometheus metrics HTTP server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{}))
	metricsAddr := ":" + cfg.MetricsPort
	metricsSrv := &http.Server{Addr: metricsAddr, Handler: metricsMux}
	go func() {
		log.Info("metrics server listening", zap.String("addr", metricsAddr))
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("metrics server error", zap.Error(err))
		}
	}()

	// Start gRPC listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.AuthServicePort))
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	// Graceful shutdown
	go func() {
		log.Info("Auth Service listening", zap.String("port", cfg.AuthServicePort))
		if err := server.Serve(listener); err != nil {
			log.Fatal("failed to serve", zap.Error(err))
		}
	}()

	health.SetReady(healthSrv, "auth")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down Auth Service...")
	health.SetNotReady(healthSrv, "auth")
	server.GracefulStop()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("metrics server shutdown error", zap.Error(err))
	}
	log.Info("Auth Service stopped")
}

package main

import (
	"XTalk/pkg/database"
	"XTalk/pkg/grpctls"
	"XTalk/pkg/health"
	"XTalk/pkg/logger"
	"XTalk/pkg/metrics"
	"XTalk/pkg/requestid"
	"XTalk/pkg/tracing"
	"XTalk/services/user/application/users/create_user"
	"XTalk/services/user/application/users/delete_user"
	"XTalk/services/user/application/users/get_user"
	"XTalk/services/user/application/users/get_user_by_email"
	"XTalk/services/user/application/users/update_status"
	"XTalk/services/user/application/users/update_user"
	"XTalk/services/user/application/validation"
	messaging "XTalk/services/user/infrastructure/messaging/rabbitmq"
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
)

func main() {
	log := logger.New()
	defer log.Sync()

	cfg := config.LoadConfig()

	// OpenTelemetry tracing
	shutdownTracer, err := tracing.Init(context.Background(), tracing.Config{
		ServiceName: "user-service",
		Endpoint:    cfg.OTELEndpoint,
	}, log)
	if err != nil {
		log.Fatal("failed to init tracing", zap.Error(err))
	}
	defer shutdownTracer(context.Background())

	// PostgreSQL
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

	// Infrastructure adapters
	userRepo := persistence.NewPostgresUserRepository(db)
	validator := validation.NewInputValidator()

	// Application handlers
	createUserHandler := create_user.NewHandler(userRepo)
	updateUserHandler := update_user.NewHandler(userRepo, validator)
	updateStatusHandler := update_status.NewHandler(userRepo)
	deleteUserHandler := delete_user.NewHandler(userRepo)
	getUserHandler := get_user.NewHandler(userRepo)
	getUserByEmailHandler := get_user_by_email.NewHandler(userRepo)

	// RabbitMQ event consumer
	eventConsumer, err := messaging.NewRabbitMQEventConsumer(cfg.RabbitMQURL, userRepo, createUserHandler, log)
	if err != nil {
		log.Warn("failed to initialize RabbitMQ consumer (event-driven updates disabled)", zap.Error(err))
	} else {
		defer eventConsumer.Close()
		if err := eventConsumer.Start(context.Background()); err != nil {
			log.Warn("failed to start RabbitMQ consumer", zap.Error(err))
		}
	}

	// gRPC service
	userService := grpcServer.NewUserGRPCService(
		createUserHandler,
		updateUserHandler,
		updateStatusHandler,
		deleteUserHandler,
		getUserHandler,
		getUserByEmailHandler,
	)

	// Prometheus gRPC metrics
	grpcMetrics := metrics.NewGRPCMetrics("user_service")

	// gRPC server options
	serverOpts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(4 * 1024 * 1024),
		grpc.StatsHandler(tracing.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			requestid.UnaryServerInterceptor(),
			metrics.UnaryServerInterceptor(grpcMetrics),
		),
	}
	if tlsCreds, err := grpctls.ServerOptions(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil {
		log.Fatal("failed to load TLS credentials", zap.Error(err))
	} else if tlsCreds != nil {
		serverOpts = append(serverOpts, tlsCreds)
		log.Info("gRPC TLS enabled")
	}

	grpcSrv := grpc.NewServer(serverOpts...)
	pb.RegisterUserServiceServer(grpcSrv, userService)

	// gRPC health check
	healthSrv := health.Register(grpcSrv)

	// Prometheus metrics HTTP server
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

	// gRPC listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Port))
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	go func() {
		log.Info("User Service listening", zap.String("port", cfg.Port))
		if err := grpcSrv.Serve(listener); err != nil {
			log.Fatal("failed to serve", zap.Error(err))
		}
	}()

	health.SetReady(healthSrv, "user")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down User Service...")
	health.SetNotReady(healthSrv, "user")
	grpcSrv.GracefulStop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("metrics server shutdown error", zap.Error(err))
	}
	log.Info("User Service stopped")
}

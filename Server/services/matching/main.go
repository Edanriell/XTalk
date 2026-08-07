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

	// Application
	"XTalk/services/matching/application/end_match"
	"XTalk/services/matching/application/get_match_history"
	"XTalk/services/matching/application/get_matching_status"
	"XTalk/services/matching/application/join_queue"
	"XTalk/services/matching/application/leave_queue"
	"XTalk/services/matching/application/services"

	// Infrastructure
	"XTalk/services/matching/adapters/external"
	"XTalk/services/matching/adapters/idgen"
	"XTalk/services/matching/adapters/messaging"
	"XTalk/services/matching/adapters/persistence"
	grpcServer "XTalk/services/matching/ports/grpc"

	// Config
	"XTalk/services/matching/config"

	// Proto
	pb "XTalk/proto/matching"

	// Shared packages
	"XTalk/pkg/database"
	"XTalk/pkg/grpctls"
	"XTalk/pkg/health"
	"XTalk/pkg/logger"
	"XTalk/pkg/metrics"
	"XTalk/pkg/requestid"
	"XTalk/pkg/tracing"
)

func main() {
	log := logger.New()
	defer log.Sync()

	cfg := config.LoadConfig()

	// Initialize OpenTelemetry tracing
	shutdownTracer, err := tracing.Init(context.Background(), tracing.Config{
		ServiceName: "matching-service",
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

	// Initialize RabbitMQ event publisher
	eventPublisher, err := messaging.NewRabbitMQEventPublisher(cfg.RabbitMQURL, "matching_events")
	if err != nil {
		log.Fatal("failed to initialize RabbitMQ", zap.Error(err))
	}
	defer func() {
		if closer, ok := eventPublisher.(interface{ Close() error }); ok {
			closer.Close()
		}
	}()

	// Create infrastructure adapters
	queueRepo := persistence.NewPostgresMatchingQueueRepository(db)
	matchHistoryRepo := persistence.NewPostgresMatchHistoryRepository(db)
	idGenerator := idgen.NewUUIDGenerator()

	chatCreator, err := external.NewGRPCChatCreator(cfg.ChatServiceAddr)
	if err != nil {
		log.Fatal("failed to initialize chat creator", zap.Error(err))
	}
	defer func() {
		if closer, ok := chatCreator.(interface{ Close() error }); ok {
			closer.Close()
		}
	}()

	userValidator, err := external.NewGRPCUserValidator(cfg.UserServiceAddr)
	if err != nil {
		log.Fatal("failed to initialize user validator", zap.Error(err))
	}
	defer func() {
		if closer, ok := userValidator.(interface{ Close() error }); ok {
			closer.Close()
		}
	}()

	// Create domain services
	matchingAlgo := services.NewMatchingAlgorithm()

	// Create application handlers
	joinQueueHandler := join_queue.NewHandler(
		queueRepo,
		matchHistoryRepo,
		userValidator,
		chatCreator,
		idGenerator,
		eventPublisher,
		matchingAlgo,
		log,
	)
	leaveQueueHandler := leave_queue.NewHandler(queueRepo, eventPublisher, log)
	endMatchHandler := end_match.NewHandler(matchHistoryRepo, eventPublisher, log)
	getMatchingStatusHandler := get_matching_status.NewHandler(queueRepo, matchHistoryRepo)
	getMatchHistoryHandler := get_match_history.NewHandler(matchHistoryRepo)

	// Create gRPC service
	matchingService := grpcServer.NewMatchingGRPCService(
		joinQueueHandler,
		leaveQueueHandler,
		endMatchHandler,
		getMatchingStatusHandler,
		getMatchHistoryHandler,
		queueRepo,
	)

	// Prometheus gRPC metrics
	grpcMetrics := metrics.NewGRPCMetrics("matching_service")

	// Create gRPC server with metrics interceptor
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
	pb.RegisterMatchingServiceServer(grpcSrv, matchingService)

	// Register gRPC health check
	healthSrv := health.Register(grpcSrv)

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
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Port))
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	go func() {
		log.Info("Matching Service listening", zap.String("port", cfg.Port))
		if err := grpcSrv.Serve(listener); err != nil {
			log.Fatal("failed to serve", zap.Error(err))
		}
	}()

	health.SetReady(healthSrv, "matching")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down Matching Service...")
	health.SetNotReady(healthSrv, "matching")
	grpcSrv.GracefulStop()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("metrics server shutdown error", zap.Error(err))
	}
	log.Info("Matching Service stopped")
}

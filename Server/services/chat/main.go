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
	"github.com/yourusername/connect/chat-service/application/commands/create_chat"
	"github.com/yourusername/connect/chat-service/application/commands/end_chat"
	"github.com/yourusername/connect/chat-service/application/queries/get_chat"
	"github.com/yourusername/connect/chat-service/application/queries/get_user_chats"

	// Infrastructure
	grpcServer "github.com/yourusername/connect/chat-service/infrastructure/grpc"
	"github.com/yourusername/connect/chat-service/infrastructure/idgen"
	"github.com/yourusername/connect/chat-service/infrastructure/messaging"
	"github.com/yourusername/connect/chat-service/infrastructure/persistence"

	// Config
	"github.com/yourusername/connect/chat-service/config"

	// Proto
	pb "github.com/yourusername/connect/proto/chat"

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
		ServiceName: "chat-service",
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

	// Create infrastructure adapters
	chatRepo := persistence.NewPostgresChatRepository(db)
	idGenerator := idgen.NewUUIDGenerator()

	// Initialize RabbitMQ event consumer
	eventConsumer, err := messaging.NewRabbitMQEventConsumer(cfg.RabbitMQURL, chatRepo, log)
	if err != nil {
		log.Warn("failed to initialize RabbitMQ consumer (event-driven activity updates disabled)", zap.Error(err))
	} else {
		defer eventConsumer.Close()
		if err := eventConsumer.Start(context.Background()); err != nil {
			log.Warn("failed to start RabbitMQ consumer", zap.Error(err))
		}
	}

	// Create application handlers
	createChatHandler := create_chat.NewHandler(chatRepo, idGenerator)
	endChatHandler := end_chat.NewHandler(chatRepo)
	getChatHandler := get_chat.NewHandler(chatRepo)
	getUserChatsHandler := get_user_chats.NewHandler(chatRepo)

	// Create gRPC service
	chatService := grpcServer.NewChatGRPCService(
		createChatHandler,
		endChatHandler,
		getChatHandler,
		getUserChatsHandler,
	)

	// Prometheus gRPC metrics
	grpcMetrics := metrics.NewGRPCMetrics("chat_service")

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
	pb.RegisterChatServiceServer(grpcSrv, chatService)

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
		log.Info("Chat Service listening", zap.String("port", cfg.Port))
		if err := grpcSrv.Serve(listener); err != nil {
			log.Fatal("failed to serve", zap.Error(err))
		}
	}()

	health.SetReady(healthSrv, "chat")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down Chat Service...")
	health.SetNotReady(healthSrv, "chat")
	grpcSrv.GracefulStop()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("metrics server shutdown error", zap.Error(err))
	}
	log.Info("Chat Service stopped")
}

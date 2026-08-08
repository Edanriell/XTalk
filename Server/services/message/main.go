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
	"XTalk/services/message/application/delete_message"
	"XTalk/services/message/application/get_messages"
	"XTalk/services/message/application/mark_as_read"
	"XTalk/services/message/application/send_message"

	// Infrastructure
	"XTalk/services/message/adapters/external"
	"XTalk/services/message/adapters/idgen"
	"XTalk/services/message/adapters/messaging"
	"XTalk/services/message/adapters/persistence"
	grpcServer "XTalk/services/message/ports/grpc"

	// Config
	"XTalk/services/message/adapters/config"

	// Proto
	pb "XTalk/proto/message"

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
		ServiceName: "message-service",
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
	eventPublisher, err := messaging.NewRabbitMQEventPublisher(cfg.RabbitMQURL, "message_events")
	if err != nil {
		log.Fatal("failed to initialize RabbitMQ", zap.Error(err))
	}
	defer func() {
		if closer, ok := eventPublisher.(interface{ Close() error }); ok {
			closer.Close()
		}
	}()

	// Create infrastructure adapters
	messageRepo := persistence.NewPostgresMessageRepository(db)
	idGenerator := idgen.NewUUIDGenerator()
	chatValidator, err := external.NewGRPCChatValidator(cfg.ChatServiceAddr)
	if err != nil {
		log.Fatal("failed to initialize chat validator", zap.Error(err))
	}
	defer func() {
		if closer, ok := chatValidator.(interface{ Close() error }); ok {
			closer.Close()
		}
	}()

	// Create application handlers
	sendMessageHandler := send_message.NewHandler(messageRepo, chatValidator, idGenerator, eventPublisher, log)
	deleteMessageHandler := delete_message.NewHandler(messageRepo, eventPublisher, log)
	markAsReadHandler := mark_as_read.NewHandler(messageRepo, eventPublisher, log)
	getMessagesHandler := get_messages.NewHandler(messageRepo, chatValidator)

	// Create gRPC service
	messageService := grpcServer.NewMessageGRPCService(
		sendMessageHandler,
		deleteMessageHandler,
		markAsReadHandler,
		getMessagesHandler,
	)

	// Prometheus gRPC metrics
	grpcMetrics := metrics.NewGRPCMetrics("message_service")

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
	pb.RegisterMessageServiceServer(grpcSrv, messageService)

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
		log.Info("Message Service listening", zap.String("port", cfg.Port))
		if err := grpcSrv.Serve(listener); err != nil {
			log.Fatal("failed to serve", zap.Error(err))
		}
	}()

	health.SetReady(healthSrv, "message")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down Message Service...")
	health.SetNotReady(healthSrv, "message")
	grpcSrv.GracefulStop()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("metrics server shutdown error", zap.Error(err))
	}
	log.Info("Message Service stopped")
}

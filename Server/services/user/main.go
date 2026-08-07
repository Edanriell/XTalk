package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"XTalk/pkg/database"
	"XTalk/pkg/grpctls"
	"XTalk/pkg/health"
	"XTalk/pkg/logger"
	"XTalk/pkg/metrics"
	"XTalk/pkg/requestid"
	"XTalk/pkg/tracing"
	userpb "XTalk/proto/user"
	userconfig "XTalk/services/user/adapters/config"
	"XTalk/services/user/adapters/messaging/rabbitmq"
	"XTalk/services/user/adapters/repositories"
	appevents "XTalk/services/user/application/events"
	userapp "XTalk/services/user/application/users"
	transportgrpc "XTalk/services/user/ports/grpc"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const shutdownTimeout = 5 * time.Second

func main() {
	log := logger.New()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	err := run(ctx, log)
	stop()
	if err != nil {
		log.Error("user service stopped with an error", zap.Error(err))
	}
	_ = log.Sync()
	if err != nil {
		os.Exit(1)
	}
}

func run(ctx context.Context, log *zap.Logger) error {
	serviceCtx, cancelService := context.WithCancel(ctx)
	defer cancelService()

	cfg := userconfig.LoadConfig()

	shutdownTracer, err := tracing.Init(ctx, tracing.Config{
		ServiceName: "user-service",
		Endpoint:    cfg.OTELEndpoint,
	}, log)
	if err != nil {
		return fmt.Errorf("initialize tracing: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := shutdownTracer(shutdownCtx); err != nil {
			log.Error("shutdown tracing", zap.Error(err))
		}
	}()

	db, err := database.Connect(database.Config{
		Host: cfg.DBHost, Port: cfg.DBPort, User: cfg.DBUser,
		Password: cfg.DBPassword, Name: cfg.DBName, SSL: cfg.DBSSL,
	}, log)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("close database", zap.Error(err))
		}
	}()

	repository := repositories.NewPostgresUserRepository(db)
	users := userapp.NewService(repository)
	eventHandler := appevents.NewHandler(users)

	consumer, err := rabbitmq.NewConsumer(cfg.RabbitMQURL, eventHandler, log)
	if err != nil {
		return fmt.Errorf("initialize integration-event consumer: %w", err)
	}
	defer func() {
		if err := consumer.Close(); err != nil {
			log.Error("close integration-event consumer", zap.Error(err))
		}
	}()
	if err := consumer.Start(serviceCtx); err != nil {
		return fmt.Errorf("start integration-event consumer: %w", err)
	}

	serverOptions := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(4 * 1024 * 1024),
		grpc.StatsHandler(tracing.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			requestid.UnaryServerInterceptor(),
			metrics.UnaryServerInterceptor(metrics.NewGRPCMetrics("user_service")),
		),
	}
	tlsOption, err := grpctls.ServerOptions(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return fmt.Errorf("load gRPC TLS credentials: %w", err)
	}
	if tlsOption != nil {
		serverOptions = append(serverOptions, tlsOption)
		log.Info("gRPC TLS enabled")
	}

	listener, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		return fmt.Errorf("listen for gRPC: %w", err)
	}

	grpcServer := grpc.NewServer(serverOptions...)
	userpb.RegisterUserServiceServer(grpcServer, transportgrpc.NewUserService(users, log))
	healthServer := health.Register(grpcServer)

	metricsServer := &http.Server{
		Addr:              ":" + cfg.MetricsPort,
		Handler:           metricsHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	grpcErrors := make(chan error, 1)
	go func() {
		log.Info("user gRPC server listening", zap.String("address", listener.Addr().String()))
		grpcErrors <- grpcServer.Serve(listener)
	}()
	metricsErrors := make(chan error, 1)
	go func() {
		log.Info("user metrics server listening", zap.String("address", metricsServer.Addr))
		metricsErrors <- metricsServer.ListenAndServe()
	}()
	health.SetReady(healthServer, "user")

	var runErr error
	select {
	case <-ctx.Done():
		log.Info("user service shutdown requested")
	case err := <-grpcErrors:
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			runErr = fmt.Errorf("serve gRPC: %w", err)
		}
	case err := <-metricsErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			runErr = fmt.Errorf("serve metrics: %w", err)
		}
	case err := <-consumer.Failures():
		runErr = fmt.Errorf("consume integration events: %w", err)
	}

	cancelService()
	health.SetNotReady(healthServer, "user")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		runErr = errors.Join(runErr, fmt.Errorf("shutdown metrics server: %w", err))
	}
	gracefulStop(shutdownCtx, grpcServer)
	log.Info("user service stopped")
	return runErr
}

func metricsHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{}))
	return mux
}

func gracefulStop(ctx context.Context, server *grpc.Server) {
	done := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		server.Stop()
	}
}

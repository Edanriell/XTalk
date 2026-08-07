package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"XTalk/gateway/circuitbreaker"
	"XTalk/gateway/config"
	"XTalk/gateway/handlers"
	"XTalk/gateway/messaging"
	"XTalk/gateway/middleware"
	"XTalk/pkg/logger"
	"XTalk/pkg/metrics"
)

func main() {
	log := logger.New()
	defer log.Sync()

	cfg := config.LoadConfig()

	// Initialize circuit breaker registry for all gRPC connections
	cbRegistry := circuitbreaker.NewRegistry(log, circuitbreaker.CBConfig{
		FailureThreshold: cfg.CBFailureThreshold,
		WindowSize:       cfg.CBWindowSize,
		Delay:            cfg.CBDelay,
		SuccessThreshold: cfg.CBSuccessThreshold,
	})

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(cfg, log, cbRegistry)
	defer authHandler.Close()
	userHandler := handlers.NewUserHandler(cfg, log, cbRegistry)
	defer userHandler.Close()
	chatHandler := handlers.NewChatHandler(cfg, log, cbRegistry)
	defer chatHandler.Close()
	messageHandler := handlers.NewMessageHandler(cfg, log, cbRegistry)
	defer messageHandler.Close()
	matchingHandler := handlers.NewMatchingHandler(cfg, log, cbRegistry)
	defer matchingHandler.Close()
	wsHandler := handlers.NewWebSocketHandler(cfg, log, cbRegistry)
	defer wsHandler.Close()

	// Initialize RabbitMQ consumer for real-time push notifications
	mqCtx, mqCancel := context.WithCancel(context.Background())
	defer mqCancel()
	mqConsumer, err := messaging.NewRabbitMQConsumer(cfg.RabbitMQURL, wsHandler, log)
	if err != nil {
		log.Warn("failed to initialize RabbitMQ consumer (real-time push disabled)", zap.Error(err))
	} else {
		defer mqConsumer.Close()
		if err := mqConsumer.Start(mqCtx); err != nil {
			log.Warn("failed to start RabbitMQ consumer", zap.Error(err))
		}
	}

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Global middleware
	httpMetrics := metrics.NewHTTPMetrics("api_gateway")
	router.Use(gin.Recovery())
	router.Use(securityHeaders())
	router.Use(middleware.Logger(log))
	router.Use(httpMetrics.GinMiddleware())
	router.Use(maxBodySize(cfg.MaxBodySize))
	router.Use(corsMiddleware(cfg.AllowedOrigins))

	// Health & metrics
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{})))

	// Public auth routes (no auth middleware) — rate-limited per IP
	authLimiter := middleware.NewRateLimiter(5, 10) // 5 req/s, burst of 10
	defer authLimiter.Close()
	auth := router.Group("/api/v1/auth")
	auth.Use(authLimiter.GinMiddleware())
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
	}

	// Protected routes
	idempotencyStore := middleware.NewIdempotencyStore(24 * time.Hour)
	defer idempotencyStore.Close()
	authMiddleware, authMiddlewareCleanup := middleware.Auth(authHandler, log)
	defer authMiddlewareCleanup()
	protected := router.Group("/api/v1")
	protected.Use(authMiddleware)
	protected.Use(middleware.Idempotency(idempotencyStore))
	{
		// Auth
		protected.POST("/auth/logout", authHandler.Logout)

		// Users
		protected.GET("/users/me", userHandler.GetCurrentUser)
		protected.PUT("/users/me", userHandler.UpdateUser)

		// Chat rooms
		protected.GET("/rooms", chatHandler.GetUserRooms)
		protected.POST("/rooms", chatHandler.CreateRoom)
		protected.GET("/rooms/:id", chatHandler.GetRoom)
		protected.DELETE("/rooms/:id", chatHandler.DeleteRoom)

		// Messages
		protected.GET("/messages", messageHandler.GetMessages)
		protected.POST("/messages", messageHandler.SendMessage)

		// Matching
		protected.POST("/matching/start", matchingHandler.StartMatching)
		protected.POST("/matching/stop", matchingHandler.StopMatching)
		protected.GET("/matching/status", matchingHandler.GetMatchingStatus)
		protected.GET("/matching/history", matchingHandler.GetMatchHistory)
		protected.POST("/matching/end", matchingHandler.EndMatch)
	}

	// WebSocket (auth handled internally)
	router.GET("/ws", wsHandler.HandleWebSocket)

	// HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.APIGatewayPort,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Graceful shutdown
	go func() {
		log.Info("API Gateway listening", zap.String("port", cfg.APIGatewayPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down API Gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("server forced to shutdown", zap.Error(err))
	}
	log.Info("API Gateway stopped")
}

func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin != "" {
			if len(allowed) == 0 {
				// No origins configured — reject cross-origin requests.
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
				c.Header("Access-Control-Max-Age", "86400")

				if c.Request.Method == "OPTIONS" {
					c.AbortWithStatus(http.StatusNoContent)
					return
				}
			} else {
				// Origin not in allow-list — reject.
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
		}

		c.Next()
	}
}

// maxBodySize limits the request body to the given number of bytes.
func maxBodySize(n int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		}
		c.Next()
	}
}

// securityHeaders adds standard security response headers to every response.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}

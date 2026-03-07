package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"XTalk/api-gateway/config"
	"XTalk/api-gateway/middlewares"
)

func main() {
	cfg := config.LoadConfig()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(cfg)
	userHandler := handlers.NewUserHandler(cfg)

	mqConsumer, err := messaging.NewRabbitMqConsumer(cfg.RabbitMqUrl, wsHandler)
	if err != nil {
		log.Printf("Warning: Failed to initialize RabbitMq consumer: %v (real-time push disabled)", err)
	} else {
		if err := mqConsumer.Start(context.Background()); err != nil {
			log.Printf("Warning: Failed to start RabbitMq consumer: %v", err)
		} else {
			defer mqConsumer.Close()
		}
	}

	// Create router
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	// Auth routes
	mux.HandleFunc("/api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("/api/v1/auth/refresh", authHandler.RefreshToken)
	mux.HandleFunc("/api/v1/auth/logout", authHandler.Logout)

	// User routes
	mux.HandleFunc("/api/v1/users/me", userHandler.GetCurrentUser)
	mux.HandleFunc("/api/v1/users/", userHandler.UpdateUser)

	// Apply middleware
	handler := middlewares.CorsMiddleware(mux)
	handler = middlewares.RateLimitMiddleware(handler)
	handler = middlewares.LoggingMiddleware(handler)

	server := &http.Server{
		Addr:         ":" + cfg.ApiGatewayPort,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("API Gateway listening on port %s", cfg.ApiGatewayPort)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

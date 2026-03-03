package main

import (
	"XTalk/api-gateway/config"
	"context"
	"log"
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
}

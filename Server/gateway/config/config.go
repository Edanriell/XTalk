package config

import (
	"log"

	"github.com/joho/godotenv"

	"XTalk/gateway/utils"
)

type Config struct {
	AuthServicePort     string
	UserServicePort     string
	ChatServicePort     string
	MessageServicePort  string
	MatchingServicePort string
	ApiGatewayPort      string
	RabbitMqUrl         string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		AuthServicePort:     utils.GetEnv("AUTH_SERVICE_PORT", "50051"),
		UserServicePort:     utils.GetEnv("USER_SERVICE_PORT", "50052"),
		ChatServicePort:     utils.GetEnv("CHAT_SERVICE_PORT", "50053"),
		MessageServicePort:  utils.GetEnv("MESSAGE_SERVICE_PORT", "50054"),
		MatchingServicePort: utils.GetEnv("MATCHING_SERVICE_PORT", "50055"),
		ApiGatewayPort:      utils.GetEnv("API_GATEWAY_PORT", "8080"),
		RabbitMqUrl:         utils.GetEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
	}
}

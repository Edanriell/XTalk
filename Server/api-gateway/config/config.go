package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
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
		AuthServicePort:     getEnv("AUTH_SERVICE_PORT", "50051"),
		UserServicePort:     getEnv("USER_SERVICE_PORT", "50052"),
		ChatServicePort:     getEnv("CHAT_SERVICE_PORT", "50053"),
		MessageServicePort:  getEnv("MESSAGE_SERVICE_PORT", "50054"),
		MatchingServicePort: getEnv("MATCHING_SERVICE_PORT", "50055"),
		ApiGatewayPort:      getEnv("API_GATEWAY_PORT", "8080"),
		RabbitMqUrl:         getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

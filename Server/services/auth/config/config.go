package config

import (
	"log"

	"github.com/joho/godotenv"

	"XTalk/gateway/utils"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSL      string

	JWTSecret        string
	JWTAccessExpiry  string
	JWTRefreshExpiry string

	RedisHost     string
	RedisPort     string
	RedisPassword string

	AuthServicePort string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		DBHost:     utils.GetEnv("DB_HOST", "localhost"),
		DBPort:     utils.GetEnv("DB_PORT", "5432"),
		DBUser:     utils.GetEnv("DB_USER", "postgres"),
		DBPassword: utils.GetEnv("DB_PASSWORD", "postgres"),
		DBName:     utils.GetEnv("DB_NAME", "connect_db"),
		DBSSL:      utils.GetEnv("DB_SSL", "disable"),

		JWTSecret:        utils.GetEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		JWTAccessExpiry:  utils.GetEnv("JWT_ACCESS_EXPIRY", "15m"),
		JWTRefreshExpiry: utils.GetEnv("JWT_REFRESH_EXPIRY", "7d"),

		RedisHost:     utils.GetEnv("REDIS_HOST", "localhost"),
		RedisPort:     utils.GetEnv("REDIS_PORT", "6379"),
		RedisPassword: utils.GetEnv("REDIS_PASSWORD", ""),

		AuthServicePort: utils.GetEnv("AUTH_SERVICE_PORT", "50051"),
	}
}

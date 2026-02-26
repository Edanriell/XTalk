package config

import (
	"XTalk/pkg/config"
)

type Config struct {
	Port        string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSL       string
	RabbitMQURL string
	MetricsPort string

	TLSCertFile string
	TLSKeyFile  string
	TLSCAFile   string

	OTELEndpoint string
}

func LoadConfig() *Config {
	return &Config{
		Port:        config.GetEnv("USER_SERVICE_PORT", "50052"),
		DBHost:      config.GetEnv("DB_HOST", "localhost"),
		DBPort:      config.GetEnv("DB_PORT", "5432"),
		DBUser:      config.GetEnv("DB_USER", "postgres"),
		DBPassword:  config.GetEnv("DB_PASSWORD", "postgres"),
		DBName:      config.GetEnv("DB_NAME", "connect_users"),
		DBSSL:       config.GetEnv("DB_SSL", "disable"),
		RabbitMQURL: config.GetEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		MetricsPort: config.GetEnv("METRICS_PORT", "9090"),

		TLSCertFile: config.GetEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:  config.GetEnv("TLS_KEY_FILE", ""),
		TLSCAFile:   config.GetEnv("TLS_CA_FILE", ""),

		OTELEndpoint: config.GetEnv("OTEL_EXPORTER_ENDPOINT", ""),
	}
}

package config

import (
	pkgcfg "XTalk/pkg/config"
)

type Config struct {
	Port            string
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	DBSSL           string
	RabbitMQURL     string
	ChatServiceAddr string
	UserServiceAddr string
	MetricsPort     string

	// TLS (optional)
	TLSCertFile string
	TLSKeyFile  string
	TLSCAFile   string

	// Observability
	OTELEndpoint string
}

func LoadConfig() *Config {
	return &Config{
		Port:            pkgcfg.GetEnv("MATCHING_SERVICE_PORT", "50055"),
		DBHost:          pkgcfg.GetEnv("DB_HOST", "localhost"),
		DBPort:          pkgcfg.GetEnv("DB_PORT", "5432"),
		DBUser:          pkgcfg.GetEnv("DB_USER", "postgres"),
		DBPassword:      pkgcfg.GetEnv("DB_PASSWORD", "postgres"),
		DBName:          pkgcfg.GetEnv("DB_NAME", "connect_matching"),
		DBSSL:           pkgcfg.GetEnv("DB_SSL", "disable"),
		RabbitMQURL:     pkgcfg.GetEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		ChatServiceAddr: pkgcfg.GetEnv("CHAT_SERVICE_ADDR", "localhost:50053"),
		UserServiceAddr: pkgcfg.GetEnv("USER_SERVICE_ADDR", "localhost:50052"),
		MetricsPort:     pkgcfg.GetEnv("METRICS_PORT", "9090"),

		TLSCertFile: pkgcfg.GetEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:  pkgcfg.GetEnv("TLS_KEY_FILE", ""),
		TLSCAFile:   pkgcfg.GetEnv("TLS_CA_FILE", ""),

		OTELEndpoint: pkgcfg.GetEnv("OTEL_EXPORTER_ENDPOINT", ""),
	}
}

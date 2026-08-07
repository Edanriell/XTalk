package config

import (
	pkgcfg "XTalk/pkg/config"
	"strconv"
	"time"
)

type Config struct {
	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSL      string

	// JWT
	JWTSecret        string
	JWTAccessExpiry  string
	JWTRefreshExpiry string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string

	// Service
	AuthServicePort string
	MetricsPort     string

	// Messaging
	RabbitMQURL string

	// Rate limiting
	GRPCRateLimit    uint
	GRPCRateWindow   time.Duration
	LoginMaxAttempts int64
	LoginWindow      time.Duration

	// TLS (optional — leave empty for insecure)
	TLSCertFile string
	TLSKeyFile  string
	TLSCAFile   string

	// Observability
	OTELEndpoint string
}

func LoadConfig() *Config {
	return &Config{
		DBHost:     pkgcfg.GetEnv("DB_HOST", "localhost"),
		DBPort:     pkgcfg.GetEnv("DB_PORT", "5432"),
		DBUser:     pkgcfg.GetEnv("DB_USER", "postgres"),
		DBPassword: pkgcfg.GetEnv("DB_PASSWORD", "postgres"),
		DBName:     pkgcfg.GetEnv("DB_NAME", "connect_db"),
		DBSSL:      pkgcfg.GetEnv("DB_SSL", "disable"),

		JWTSecret:        pkgcfg.GetEnv("JWT_SECRET", ""),
		JWTAccessExpiry:  pkgcfg.GetEnv("JWT_ACCESS_EXPIRY", "15m"),
		JWTRefreshExpiry: pkgcfg.GetEnv("JWT_REFRESH_EXPIRY", "168h"),

		RedisHost:     pkgcfg.GetEnv("REDIS_HOST", "localhost"),
		RedisPort:     pkgcfg.GetEnv("REDIS_PORT", "6379"),
		RedisPassword: pkgcfg.GetEnv("REDIS_PASSWORD", ""),

		AuthServicePort: pkgcfg.GetEnv("AUTH_SERVICE_PORT", "50051"),
		MetricsPort:     pkgcfg.GetEnv("METRICS_PORT", "9090"),

		RabbitMQURL: pkgcfg.GetEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),

		GRPCRateLimit:    parseUint(pkgcfg.GetEnv("GRPC_RATE_LIMIT", "60"), 60),
		GRPCRateWindow:   parseDuration(pkgcfg.GetEnv("GRPC_RATE_WINDOW", "1m"), time.Minute),
		LoginMaxAttempts: parseInt64(pkgcfg.GetEnv("LOGIN_MAX_ATTEMPTS", "5"), 5),
		LoginWindow:      parseDuration(pkgcfg.GetEnv("LOGIN_WINDOW", "15m"), 15*time.Minute),

		TLSCertFile: pkgcfg.GetEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:  pkgcfg.GetEnv("TLS_KEY_FILE", ""),
		TLSCAFile:   pkgcfg.GetEnv("TLS_CA_FILE", ""),

		OTELEndpoint: pkgcfg.GetEnv("OTEL_EXPORTER_ENDPOINT", ""),
	}
}

func parseDuration(raw string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return d
}

func parseUint(raw string, fallback uint) uint {
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return uint(v)
}

func parseInt64(raw string, fallback int64) int64 {
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return v
}

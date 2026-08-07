package config

import (
	"log"
	"strconv"
	"strings"
	"time"

	cfg "XTalk/pkg/config"
)

type Config struct {
	// Service ports
	AuthServicePort     string
	UserServicePort     string
	ChatServicePort     string
	MessageServicePort  string
	MatchingServicePort string
	APIGatewayPort      string

	// Service addresses (for Docker/k8s service discovery)
	AuthServiceAddr     string
	UserServiceAddr     string
	ChatServiceAddr     string
	MessageServiceAddr  string
	MatchingServiceAddr string

	// Messaging
	RabbitMQURL string

	// Security
	AllowedOrigins []string

	// Timeouts
	GRPCTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// Circuit breaker
	CBFailureThreshold int
	CBWindowSize       int
	CBDelay            time.Duration
	CBSuccessThreshold int

	// WebSocket
	WSReadBufferSize  int
	WSWriteBufferSize int

	// Request limits
	MaxBodySize int64 // maximum request body in bytes
}

func LoadConfig() *Config {
	authPort := cfg.GetEnv("AUTH_SERVICE_PORT", "50051")
	userPort := cfg.GetEnv("USER_SERVICE_PORT", "50052")
	chatPort := cfg.GetEnv("CHAT_SERVICE_PORT", "50053")
	msgPort := cfg.GetEnv("MESSAGE_SERVICE_PORT", "50054")
	matchPort := cfg.GetEnv("MATCHING_SERVICE_PORT", "50055")

	return &Config{
		AuthServicePort:     authPort,
		UserServicePort:     userPort,
		ChatServicePort:     chatPort,
		MessageServicePort:  msgPort,
		MatchingServicePort: matchPort,
		APIGatewayPort:      cfg.GetEnv("API_GATEWAY_PORT", "8080"),

		AuthServiceAddr:     ensureGRPCScheme(cfg.GetEnv("AUTH_SERVICE_ADDR", "localhost:"+authPort)),
		UserServiceAddr:     ensureGRPCScheme(cfg.GetEnv("USER_SERVICE_ADDR", "localhost:"+userPort)),
		ChatServiceAddr:     ensureGRPCScheme(cfg.GetEnv("CHAT_SERVICE_ADDR", "localhost:"+chatPort)),
		MessageServiceAddr:  ensureGRPCScheme(cfg.GetEnv("MESSAGE_SERVICE_ADDR", "localhost:"+msgPort)),
		MatchingServiceAddr: ensureGRPCScheme(cfg.GetEnv("MATCHING_SERVICE_ADDR", "localhost:"+matchPort)),

		RabbitMQURL: cfg.GetEnv("RABBITMQ_URL", "amqp://localhost:5672/"),

		AllowedOrigins: parseOrigins(cfg.GetEnv("ALLOWED_ORIGINS", "")),

		GRPCTimeout: parseDuration(cfg.GetEnv("GRPC_TIMEOUT", "5s"), 5*time.Second),

		ReadTimeout:  parseDuration(cfg.GetEnv("HTTP_READ_TIMEOUT", "15s"), 15*time.Second),
		WriteTimeout: parseDuration(cfg.GetEnv("HTTP_WRITE_TIMEOUT", "15s"), 15*time.Second),
		IdleTimeout:  parseDuration(cfg.GetEnv("HTTP_IDLE_TIMEOUT", "60s"), 60*time.Second),

		CBFailureThreshold: parseInt(cfg.GetEnv("CB_FAILURE_THRESHOLD", "6"), 6),
		CBWindowSize:       parseInt(cfg.GetEnv("CB_WINDOW_SIZE", "10"), 10),
		CBDelay:            parseDuration(cfg.GetEnv("CB_DELAY", "30s"), 30*time.Second),
		CBSuccessThreshold: parseInt(cfg.GetEnv("CB_SUCCESS_THRESHOLD", "3"), 3),

		WSReadBufferSize:  parseInt(cfg.GetEnv("WS_READ_BUFFER_SIZE", "4096"), 4096),
		WSWriteBufferSize: parseInt(cfg.GetEnv("WS_WRITE_BUFFER_SIZE", "4096"), 4096),

		MaxBodySize: int64(parseInt(cfg.GetEnv("MAX_BODY_SIZE", "1048576"), 1048576)), // 1 MB default
	}
}

func parseDuration(raw string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("WARN: invalid duration %q, using default %s", raw, fallback)
		return fallback
	}
	return d
}

func parseInt(raw string, fallback int) int {
	v, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("WARN: invalid integer %q, using default %d", raw, fallback)
		return fallback
	}
	return v
}

// ensureGRPCScheme prepends "passthrough:///" to the address if it doesn't
// already have a scheme. grpc.NewClient (grpc-go v1.63+) requires a URI scheme;
// passthrough delegates DNS resolution to the OS (e.g. Docker/k8s DNS).
func ensureGRPCScheme(addr string) string {
	if strings.Contains(addr, "://") {
		return addr
	}
	return "passthrough:///" + addr
}

func parseOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

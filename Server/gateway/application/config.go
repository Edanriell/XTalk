package application

import "time"

// Config contains values required to compose the gateway. Loading these values
// from environment variables belongs to an adapter.
type Config struct {
	AuthServicePort     string
	UserServicePort     string
	ChatServicePort     string
	MessageServicePort  string
	MatchingServicePort string
	APIGatewayPort      string

	AuthServiceAddr     string
	UserServiceAddr     string
	ChatServiceAddr     string
	MessageServiceAddr  string
	MatchingServiceAddr string

	RabbitMQURL    string
	AllowedOrigins []string

	GRPCTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	CBFailureThreshold int
	CBWindowSize       int
	CBDelay            time.Duration
	CBSuccessThreshold int

	WSReadBufferSize  int
	WSWriteBufferSize int
	MaxBodySize       int64
}

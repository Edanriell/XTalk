package rabbitmq

import (
	"fmt"
	"math"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const (
	maxRetries     = 10
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
	heartbeat      = 10 * time.Second
)

// Connect dials RabbitMQ with retry and exponential backoff. Caller owns both resources.
func Connect(url string, log *zap.Logger) (*amqp091.Connection, *amqp091.Channel, error) {
	cfg := amqp091.Config{
		Heartbeat: heartbeat,
	}

	var conn *amqp091.Connection
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		conn, err = amqp091.DialConfig(url, cfg)
		if err == nil {
			break
		}

		backoff := time.Duration(math.Min(
			float64(initialBackoff)*math.Pow(2, float64(attempt)),
			float64(maxBackoff),
		))
		log.Warn("failed to connect to RabbitMQ, retrying",
			zap.Int("attempt", attempt+1),
			zap.Duration("backoff", backoff),
			zap.Error(err),
		)
		time.Sleep(backoff)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("dial rabbitmq after %d retries: %w", maxRetries, err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("open channel: %w", err)
	}

	log.Info("connected to RabbitMQ")
	return conn, ch, nil
}

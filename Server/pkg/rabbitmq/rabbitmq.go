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

// ConnectWithReconnect behaves like Connect but also returns a channel that
// emits a new *amqp091.Connection each time the previous one is lost.
// The caller should select on the returned channel and re-initialise
// consumers/publishers when a new connection arrives.
func ConnectWithReconnect(url string, log *zap.Logger) (*amqp091.Connection, *amqp091.Channel, <-chan *amqp091.Connection, error) {
	conn, ch, err := Connect(url, log)
	if err != nil {
		return nil, nil, nil, err
	}

	reconnCh := make(chan *amqp091.Connection)

	go func() {
		defer close(reconnCh)
		for {
			reason, ok := <-conn.NotifyClose(make(chan *amqp091.Error))
			if !ok {
				log.Info("RabbitMQ connection closed cleanly")
				return
			}
			log.Warn("RabbitMQ connection lost, reconnecting", zap.Any("reason", reason))

			// Retry reconnection indefinitely with back-off (Connect already retries internally).
			for {
				newConn, _, reconnErr := Connect(url, log)
				if reconnErr != nil {
					log.Error("failed to reconnect to RabbitMQ, will retry in 30s", zap.Error(reconnErr))
					time.Sleep(30 * time.Second)
					continue
				}
				conn = newConn
				reconnCh <- newConn
				break
			}
		}
	}()

	return conn, ch, reconnCh, nil
}

// DeclareTopicExchange declares a durable topic exchange.
func DeclareTopicExchange(ch *amqp091.Channel, exchange string) error {
	return ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil)
}

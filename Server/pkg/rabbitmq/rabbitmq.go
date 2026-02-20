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

func Connect(url string, log *zap.Logger) (*amqp091.Connection, *amqp091.Channel, error) {
	cfg := amqp091.Config{
		Heartbeat: heartbeat,
	}

	var connection *amqp091.Connection
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		connection, err = amqp091.DialConfig(url, cfg)
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

	ch, err := connection.Channel()
	if err != nil {
		connection.Close()
		return nil, nil, fmt.Errorf("open channel: %w", err)
	}

	log.Info("connected to RabbitMQ")

	return connection, ch, nil
}

func ConnectWithReconnect(url string, log *zap.Logger) (*amqp091.Connection, *amqp091.Channel, <-chan *amqp091.Connection, error) {
	connection, ch, err := Connect(url, log)
	if err != nil {
		return nil, nil, nil, err
	}

	reConnectionCh := make(chan *amqp091.Connection)

	go func() {
		defer close(reConnectionCh)

		for {
			reason, ok := <-connection.NotifyClose(make(chan *amqp091.Error))
			if !ok {
				log.Info("RabbitMQ connection closed cleanly")
				return
			}
			log.Warn("RabbitMQ connection lost, reconnecting", zap.Any("reason", reason))

			for {
				newConnection, _, reConnectionErr := Connect(url, log)

				if reConnectionErr != nil {
					log.Error("failed to reconnect to RabbitMQ, will retry in 30s", zap.Error(reConnectionErr))
					time.Sleep(30 * time.Second)
					continue
				}

				connection = newConnection
				reConnectionCh <- newConnection

				break
			}
		}
	}()

	return connection, ch, reConnectionCh, nil
}

func DeclareTopicExchange(ch *amqp091.Channel, exchange string) error {
	return ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil)
}

package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/rabbitmq/amqp091-go"
)

// RabbitMQEventPublisher implements EventPublisher using RabbitMQ.
type RabbitMQEventPublisher struct {
	conn     *amqp091.Connection
	channel  *amqp091.Channel
	exchange string
	mu       sync.Mutex
}

// NewRabbitMQEventPublisher creates a new RabbitMQ event publisher for auth events.
func NewRabbitMQEventPublisher(rabbitURL, exchange string) (interfaces.EventPublisher, error) {
	conn, err := amqp091.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	err = channel.ExchangeDeclare(
		exchange, "topic", true, false, false, false, nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	return &RabbitMQEventPublisher{
		conn:     conn,
		channel:  channel,
		exchange: exchange,
	}, nil
}

// PublishUserRegistered publishes a user.registered event.
func (p *RabbitMQEventPublisher) PublishUserRegistered(ctx context.Context, event interfaces.UserRegisteredEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	return p.channel.PublishWithContext(
		ctx,
		p.exchange,
		"auth.user_registered",
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp091.Persistent,
		},
	)
}

// Close closes the RabbitMQ connection.
func (p *RabbitMQEventPublisher) Close() error {
	if err := p.channel.Close(); err != nil {
		return err
	}
	return p.conn.Close()
}

package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"XTalk/services/message/application/interfaces"
	"XTalk/services/message/domain/events"
	"github.com/rabbitmq/amqp091-go"
)

// RabbitMQEventPublisher implements EventPublisher using RabbitMQ
type RabbitMQEventPublisher struct {
	conn     *amqp091.Connection
	channel  *amqp091.Channel
	exchange string
	mu       sync.Mutex // amqp091.Channel is NOT goroutine-safe
}

// NewRabbitMQEventPublisher creates a new RabbitMQ event publisher
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

	// Declare exchange
	err = channel.ExchangeDeclare(
		exchange, // name
		"topic",  // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
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

// PublishMessageSent publishes a message sent event
func (p *RabbitMQEventPublisher) PublishMessageSent(ctx context.Context, event events.MessageSentEvent) error {
	return p.publish(ctx, "message.sent", event)
}

// PublishMessageRead publishes a message read event
func (p *RabbitMQEventPublisher) PublishMessageRead(ctx context.Context, event events.MessageReadEvent) error {
	return p.publish(ctx, "message.read", event)
}

// PublishMessageDeleted publishes a message deleted event
func (p *RabbitMQEventPublisher) PublishMessageDeleted(ctx context.Context, event events.MessageDeletedEvent) error {
	return p.publish(ctx, "message.deleted", event)
}

// publish is a helper method to publish events
func (p *RabbitMQEventPublisher) publish(ctx context.Context, routingKey string, event interface{}) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	err = p.channel.PublishWithContext(
		ctx,
		p.exchange, // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp091.Persistent,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// Close closes the RabbitMQ connection
func (p *RabbitMQEventPublisher) Close() error {
	if err := p.channel.Close(); err != nil {
		return err
	}
	return p.conn.Close()
}

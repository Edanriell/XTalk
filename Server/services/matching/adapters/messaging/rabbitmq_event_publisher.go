package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"XTalk/services/matching/application/interfaces"
	"XTalk/services/matching/domain/events"
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

// PublishUserJoinedQueue publishes a user joined queue event
func (p *RabbitMQEventPublisher) PublishUserJoinedQueue(ctx context.Context, event events.UserJoinedQueueEvent) error {
	return p.publish(ctx, "matching.user_joined_queue", event)
}

// PublishUserLeftQueue publishes a user left queue event
func (p *RabbitMQEventPublisher) PublishUserLeftQueue(ctx context.Context, event events.UserLeftQueueEvent) error {
	return p.publish(ctx, "matching.user_left_queue", event)
}

// PublishMatchFound publishes a match found event
func (p *RabbitMQEventPublisher) PublishMatchFound(ctx context.Context, event events.MatchFoundEvent) error {
	return p.publish(ctx, "matching.match_found", event)
}

// PublishMatchCompleted publishes a match completed event
func (p *RabbitMQEventPublisher) PublishMatchCompleted(ctx context.Context, event events.MatchCompletedEvent) error {
	return p.publish(ctx, "matching.match_completed", event)
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

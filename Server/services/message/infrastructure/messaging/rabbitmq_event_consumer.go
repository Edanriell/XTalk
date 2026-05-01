package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"github.com/yourusername/connect/chat-service/domain/repositories"
)

// MessageSentEvent mirrors message-service domain event
type MessageSentEvent struct {
	MessageID   string    `json:"MessageID"`
	ChatID      string    `json:"ChatID"`
	SenderID    string    `json:"SenderID"`
	RecipientID string    `json:"RecipientID"`
	Content     string    `json:"Content"`
	MessageType string    `json:"MessageType"`
	Timestamp   time.Time `json:"Timestamp"`
}

// RabbitMQEventConsumer consumes message events to update chat activity
type RabbitMQEventConsumer struct {
	conn     *amqp091.Connection
	channel  *amqp091.Channel
	chatRepo repositories.ChatRepository
	log      *zap.Logger
	done     chan struct{}
}

// NewRabbitMQEventConsumer creates a new RabbitMQ event consumer for chat-service
func NewRabbitMQEventConsumer(rabbitURL string, chatRepo repositories.ChatRepository, log *zap.Logger) (*RabbitMQEventConsumer, error) {
	conn, err := amqp091.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return &RabbitMQEventConsumer{
		conn:     conn,
		channel:  channel,
		chatRepo: chatRepo,
		log:      log,
		done:     make(chan struct{}),
	}, nil
}

// Start begins consuming message events
func (c *RabbitMQEventConsumer) Start(ctx context.Context) error {
	exchange := "message_events"
	queueName := "chat-service.messages"

	// Declare exchange (must match publisher)
	err := c.channel.ExchangeDeclare(
		exchange, "topic", true, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	q, err := c.channel.QueueDeclare(
		queueName, true, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind to message.sent
	if err := c.channel.QueueBind(q.Name, "message.sent", exchange, false, nil); err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// Start consuming
	msgs, err := c.channel.Consume(
		q.Name, "", false, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	go func() {
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				if err := c.handleMessageSent(ctx, msg.Body); err != nil {
					c.log.Error("failed to handle message.sent, nacking", zap.Error(err))
					msg.Nack(false, true) // requeue
				} else {
					msg.Ack(false)
				}
			case <-c.done:
				return
			}
		}
	}()

	c.log.Info("RabbitMQ consumer started for chat-service", zap.String("routing_key", "message.sent"))
	return nil
}

// handleMessageSent updates the chat's last activity timestamp when a message is sent
func (c *RabbitMQEventConsumer) handleMessageSent(ctx context.Context, body []byte) error {
	var event MessageSentEvent
	if err := json.Unmarshal(body, &event); err != nil {
		c.log.Error("failed to unmarshal message.sent event", zap.Error(err))
		return nil // bad payload, don't requeue
	}

	// Find the chat and update its activity
	chat, err := c.chatRepo.FindByID(ctx, event.ChatID)
	if err != nil {
		c.log.Error("failed to find chat for activity update", zap.String("chat_id", event.ChatID), zap.Error(err))
		return err
	}

	chat.UpdateActivity()

	if err := c.chatRepo.Save(ctx, chat); err != nil {
		c.log.Error("failed to save chat after activity update", zap.String("chat_id", event.ChatID), zap.Error(err))
		return err
	}

	c.log.Info("updated chat last activity", zap.String("chat_id", event.ChatID), zap.String("sender_id", event.SenderID))
	return nil
}

// Close gracefully shuts down the consumer
func (c *RabbitMQEventConsumer) Close() error {
	close(c.done)
	if err := c.channel.Close(); err != nil {
		return err
	}
	return c.conn.Close()
}

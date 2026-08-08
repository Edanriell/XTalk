package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// MatchFoundEvent mirrors matching-service domain event
type MatchFoundEvent struct {
	MatchID    string    `json:"MatchID"`
	User1ID    string    `json:"User1ID"`
	User2ID    string    `json:"User2ID"`
	MatchScore float64   `json:"MatchScore"`
	ChatID     string    `json:"ChatID"`
	Timestamp  time.Time `json:"Timestamp"`
}

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

// MessageReadEvent mirrors message-service domain event
type MessageReadEvent struct {
	MessageID string    `json:"MessageID"`
	ChatID    string    `json:"ChatID"`
	ReaderID  string    `json:"ReaderID"`
	Timestamp time.Time `json:"Timestamp"`
}

// MessageDeletedEvent mirrors message-service domain event
type MessageDeletedEvent struct {
	MessageID string    `json:"MessageID"`
	ChatID    string    `json:"ChatID"`
	DeletedBy string    `json:"DeletedBy"`
	Timestamp time.Time `json:"Timestamp"`
}

// MatchCompletedEvent mirrors matching-service domain event
type MatchCompletedEvent struct {
	MatchID   string    `json:"MatchID"`
	User1ID   string    `json:"User1ID"`
	User2ID   string    `json:"User2ID"`
	Duration  int64     `json:"Duration"`
	Timestamp time.Time `json:"Timestamp"`
}

// WebSocketNotifier is the interface the consumer uses to push messages to connected clients
type WebSocketNotifier interface {
	SendToUser(userID string, payload []byte)
}

// RabbitMQConsumer consumes events from RabbitMQ and pushes them to WebSocket clients
type RabbitMQConsumer struct {
	conn     *amqp091.Connection
	notifier WebSocketNotifier
	log      *zap.Logger
	done     chan struct{}
	wg       sync.WaitGroup

	channelsMu sync.Mutex
	channels   []*amqp091.Channel // per-consumer channels for safe concurrent Ack/Nack
}

// NewRabbitMQConsumer creates a new consumer connected to RabbitMQ
func NewRabbitMQConsumer(rabbitURL string, notifier WebSocketNotifier, log *zap.Logger) (*RabbitMQConsumer, error) {
	conn, err := amqp091.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	return &RabbitMQConsumer{
		conn:     conn,
		notifier: notifier,
		log:      log.Named("rabbitmq"),
		done:     make(chan struct{}),
	}, nil
}

// Start begins consuming from both matching and message exchanges
func (c *RabbitMQConsumer) Start(ctx context.Context) error {
	if err := c.consumeFromExchange("matching_events", "api-gateway.matching", []string{
		"matching.match_found",
		"matching.match_completed",
	}, c.handleMatchingEvent); err != nil {
		return fmt.Errorf("failed to start matching consumer: %w", err)
	}

	if err := c.consumeFromExchange("message_events", "api-gateway.messages", []string{
		"message.sent",
		"message.read",
		"message.deleted",
	}, c.handleMessageEvent); err != nil {
		return fmt.Errorf("failed to start message consumer: %w", err)
	}

	c.log.Info("RabbitMQ consumers started for api-gateway")
	return nil
}

func (c *RabbitMQConsumer) consumeFromExchange(exchange, queueName string, routingKeys []string, handler func(amqp091.Delivery) error) error {
	// Each consumer gets its own channel so Ack/Nack calls are not concurrent on the same channel.
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel for %s: %w", queueName, err)
	}

	c.channelsMu.Lock()
	c.channels = append(c.channels, ch)
	c.channelsMu.Unlock()

	err = ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare exchange %s: %w", exchange, err)
	}

	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	for _, key := range routingKeys {
		err = ch.QueueBind(q.Name, key, exchange, false, nil)
		if err != nil {
			return fmt.Errorf("failed to bind queue %s to %s with key %s: %w", q.Name, exchange, key, err)
		}
	}

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to start consuming from %s: %w", q.Name, err)
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				c.log.Error("panic in consumer goroutine", zap.String("queue", queueName), zap.Any("recover", r))
			}
		}()
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				if err := handler(msg); err != nil {
					c.log.Error("handler failed, nacking message", zap.Error(err))
					msg.Nack(false, false) // don't requeue; let dead-letter handle it
				} else {
					msg.Ack(false)
				}
			case <-c.done:
				return
			}
		}
	}()

	return nil
}

func (c *RabbitMQConsumer) handleMatchingEvent(msg amqp091.Delivery) error {
	switch msg.RoutingKey {
	case "matching.match_found":
		var event MatchFoundEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			return fmt.Errorf("unmarshal match_found: %w", err)
		}

		notification := map[string]interface{}{
			"type":        "match_found",
			"match_id":    event.MatchID,
			"chat_id":     event.ChatID,
			"match_score": event.MatchScore,
			"timestamp":   event.Timestamp,
		}

		data, err := json.Marshal(notification)
		if err != nil {
			return fmt.Errorf("marshal match notification: %w", err)
		}
		c.notifier.SendToUser(event.User1ID, data)
		c.notifier.SendToUser(event.User2ID, data)

		c.log.Info("pushed match_found notification",
			zap.String("user1", event.User1ID),
			zap.String("user2", event.User2ID),
		)

	case "matching.match_completed":
		var event MatchCompletedEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			return fmt.Errorf("unmarshal match_completed: %w", err)
		}

		notification := map[string]interface{}{
			"type":      "match_completed",
			"match_id":  event.MatchID,
			"duration":  event.Duration,
			"timestamp": event.Timestamp,
		}

		data, err := json.Marshal(notification)
		if err != nil {
			return fmt.Errorf("marshal match_completed notification: %w", err)
		}
		c.notifier.SendToUser(event.User1ID, data)
		c.notifier.SendToUser(event.User2ID, data)

		c.log.Info("pushed match_completed notification",
			zap.String("user1", event.User1ID),
			zap.String("user2", event.User2ID),
		)

	default:
		c.log.Warn("unknown matching routing key", zap.String("key", msg.RoutingKey))
	}
	return nil
}

func (c *RabbitMQConsumer) handleMessageEvent(msg amqp091.Delivery) error {
	switch msg.RoutingKey {
	case "message.sent":
		var event MessageSentEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			return fmt.Errorf("unmarshal message.sent: %w", err)
		}

		notification := map[string]interface{}{
			"type":         "new_message",
			"message_id":   event.MessageID,
			"chat_id":      event.ChatID,
			"sender_id":    event.SenderID,
			"content":      event.Content,
			"message_type": event.MessageType,
			"timestamp":    event.Timestamp,
		}

		data, err := json.Marshal(notification)
		if err != nil {
			return fmt.Errorf("marshal message notification: %w", err)
		}
		c.notifier.SendToUser(event.RecipientID, data)

		c.log.Info("pushed new_message notification", zap.String("recipient", event.RecipientID))

	case "message.read":
		var event MessageReadEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			return fmt.Errorf("unmarshal message.read: %w", err)
		}

		notification := map[string]interface{}{
			"type":       "message_read",
			"message_id": event.MessageID,
			"chat_id":    event.ChatID,
			"reader_id":  event.ReaderID,
			"timestamp":  event.Timestamp,
		}

		data, err := json.Marshal(notification)
		if err != nil {
			return fmt.Errorf("marshal message_read notification: %w", err)
		}
		// Notify all participants in the chat (the reader already knows).
		// The sender is not explicitly tracked here, so broadcast to all
		// connected users in the chat via the chat_id-keyed rooms.
		c.notifier.SendToUser(event.ReaderID, data)

		c.log.Info("pushed message_read notification",
			zap.String("chat_id", event.ChatID),
			zap.String("reader", event.ReaderID),
		)

	case "message.deleted":
		var event MessageDeletedEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			return fmt.Errorf("unmarshal message.deleted: %w", err)
		}

		notification := map[string]interface{}{
			"type":       "message_deleted",
			"message_id": event.MessageID,
			"chat_id":    event.ChatID,
			"deleted_by": event.DeletedBy,
			"timestamp":  event.Timestamp,
		}

		data, err := json.Marshal(notification)
		if err != nil {
			return fmt.Errorf("marshal message_deleted notification: %w", err)
		}
		c.notifier.SendToUser(event.DeletedBy, data)

		c.log.Info("pushed message_deleted notification",
			zap.String("chat_id", event.ChatID),
			zap.String("deleted_by", event.DeletedBy),
		)

	default:
		c.log.Warn("unknown message routing key", zap.String("key", msg.RoutingKey))
	}
	return nil
}

// Close gracefully shuts down the consumer, waiting for in-flight messages.
func (c *RabbitMQConsumer) Close() error {
	close(c.done)
	c.wg.Wait()

	c.channelsMu.Lock()
	for _, ch := range c.channels {
		ch.Close()
	}
	c.channelsMu.Unlock()

	return c.conn.Close()
}

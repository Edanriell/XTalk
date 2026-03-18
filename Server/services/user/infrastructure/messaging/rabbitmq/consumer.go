package messaging

import (
	"XTalk/services/user/application/users/create_user"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// --- event payloads ---

type UserRegisteredEvent struct {
	UserID   string `json:"UserID"`
	Username string `json:"Username"`
	Email    string `json:"Email"`
}

type MatchFoundEvent struct {
	MatchID    string    `json:"MatchID"`
	User1ID    string    `json:"User1ID"`
	User2ID    string    `json:"User2ID"`
	MatchScore float64   `json:"MatchScore"`
	ChatID     string    `json:"ChatID"`
	Timestamp  time.Time `json:"Timestamp"`
}

type MatchCompletedEvent struct {
	MatchID   string    `json:"MatchID"`
	User1ID   string    `json:"User1ID"`
	User2ID   string    `json:"User2ID"`
	Duration  int64     `json:"Duration"`
	Timestamp time.Time `json:"Timestamp"`
}

// RabbitMQEventConsumer consumes domain events relevant to user-service.
type RabbitMQEventConsumer struct {
	conn              *amqp091.Connection
	channel           *amqp091.Channel
	userRepo          repositories.UserRepository
	createUserHandler *create_user.Handler
	log               *zap.Logger
	done              chan struct{}
}

func NewRabbitMQEventConsumer(
	rabbitURL string,
	userRepo repositories.UserRepository,
	createUserHandler *create_user.Handler,
	log *zap.Logger,
) (*RabbitMQEventConsumer, error) {
	conn, err := amqp091.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	return &RabbitMQEventConsumer{
		conn:              conn,
		channel:           ch,
		userRepo:          userRepo,
		createUserHandler: createUserHandler,
		log:               log,
		done:              make(chan struct{}),
	}, nil
}

func (c *RabbitMQEventConsumer) Start(ctx context.Context) error {
	const queueName = "user-service.events"

	for _, exchange := range []string{"auth_events", "matching_events"} {
		if err := c.channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
			return fmt.Errorf("declare exchange %s: %w", exchange, err)
		}
	}

	q, err := c.channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	bindings := []struct{ key, exchange string }{
		{"auth.user_registered", "auth_events"},
		{"matching.match_found", "matching_events"},
		{"matching.match_completed", "matching_events"},
	}
	for _, b := range bindings {
		if err := c.channel.QueueBind(q.Name, b.key, b.exchange, false, nil); err != nil {
			return fmt.Errorf("bind %s: %w", b.key, err)
		}
	}

	msgs, err := c.channel.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("start consuming: %w", err)
	}

	go func() {
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				if err := c.handleMessage(ctx, msg); err != nil {
					c.log.Error("failed to handle message, nacking",
						zap.String("routing_key", msg.RoutingKey), zap.Error(err))
					msg.Nack(false, true)
				} else {
					msg.Ack(false)
				}
			case <-c.done:
				return
			}
		}
	}()

	c.log.Info("RabbitMQ consumer started for user-service",
		zap.Strings("routing_keys", []string{
			"auth.user_registered",
			"matching.match_found",
			"matching.match_completed",
		}))
	return nil
}

func (c *RabbitMQEventConsumer) handleMessage(ctx context.Context, msg amqp091.Delivery) error {
	switch msg.RoutingKey {
	case "auth.user_registered":
		return c.handleUserRegistered(ctx, msg.Body)
	case "matching.match_found":
		return c.handleMatchFound(ctx, msg.Body)
	case "matching.match_completed":
		return c.handleMatchCompleted(ctx, msg.Body)
	default:
		c.log.Warn("unknown routing key", zap.String("routing_key", msg.RoutingKey))
		return nil
	}
}

func (c *RabbitMQEventConsumer) handleUserRegistered(ctx context.Context, body []byte) error {
	var event UserRegisteredEvent
	if err := json.Unmarshal(body, &event); err != nil {
		c.log.Error("unmarshal user_registered event", zap.Error(err))
		return nil // bad payload — don't requeue
	}

	if err := c.createUserHandler.Handle(ctx, create_user.Command{
		UserID:   event.UserID,
		Username: event.Username,
		Email:    event.Email,
	}); err != nil {
		c.log.Error("create user profile", zap.String("user_id", event.UserID), zap.Error(err))
		return err
	}

	c.log.Info("created user profile from registration event",
		zap.String("user_id", event.UserID), zap.String("username", event.Username))
	return nil
}

func (c *RabbitMQEventConsumer) handleMatchFound(ctx context.Context, body []byte) error {
	var event MatchFoundEvent
	if err := json.Unmarshal(body, &event); err != nil {
		c.log.Error("unmarshal match_found event", zap.Error(err))
		return nil
	}

	var lastErr error
	for _, uid := range []string{event.User1ID, event.User2ID} {
		if err := c.userRepo.UpdateStatus(ctx, uid, valueobjects.StatusAway); err != nil {
			c.log.Error("update user status on match_found", zap.String("user_id", uid), zap.Error(err))
			lastErr = err
		}
	}
	if lastErr != nil {
		return lastErr
	}

	c.log.Info("updated users to away status",
		zap.String("match_id", event.MatchID),
		zap.String("user1", event.User1ID), zap.String("user2", event.User2ID))
	return nil
}

func (c *RabbitMQEventConsumer) handleMatchCompleted(ctx context.Context, body []byte) error {
	var event MatchCompletedEvent
	if err := json.Unmarshal(body, &event); err != nil {
		c.log.Error("unmarshal match_completed event", zap.Error(err))
		return nil
	}

	var lastErr error
	for _, uid := range []string{event.User1ID, event.User2ID} {
		if err := c.userRepo.UpdateStatus(ctx, uid, valueobjects.StatusOnline); err != nil {
			c.log.Error("update user status on match_completed", zap.String("user_id", uid), zap.Error(err))
			lastErr = err
		}
	}
	if lastErr != nil {
		return lastErr
	}

	c.log.Info("updated users to online status",
		zap.String("match_id", event.MatchID),
		zap.String("user1", event.User1ID), zap.String("user2", event.User2ID))
	return nil
}

func (c *RabbitMQEventConsumer) Close() error {
	close(c.done)
	if err := c.channel.Close(); err != nil {
		return err
	}
	return c.conn.Close()
}

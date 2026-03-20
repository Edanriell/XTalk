package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	appevents "XTalk/services/user/application/events"

	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const queueName = "user-service.events"

type eventHandler interface {
	UserRegistered(context.Context, appevents.UserRegistered) error
	MatchFound(context.Context, appevents.MatchFound) error
	MatchCompleted(context.Context, appevents.MatchCompleted) error
}

type binding struct {
	exchange   string
	routingKey string
}

var bindings = []binding{
	{exchange: "auth_events", routingKey: "auth.user_registered"},
	{exchange: "matching_events", routingKey: "matching.match_found"},
	{exchange: "matching_events", routingKey: "matching.match_completed"},
}

// Consumer is a RabbitMQ adapter for external integration events.
type Consumer struct {
	connection *amqp091.Connection
	channel    *amqp091.Channel
	handler    eventHandler
	log        *zap.Logger
	closeOnce  sync.Once
	failures   chan error
}

func NewConsumer(url string, handler eventHandler, log *zap.Logger) (*Consumer, error) {
	connection, err := amqp091.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connect to RabbitMQ: %w", err)
	}
	channel, err := connection.Channel()
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("open RabbitMQ channel: %w", err)
	}
	return &Consumer{
		connection: connection, channel: channel, handler: handler, log: log,
		failures: make(chan error, 1),
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	queue, err := c.declareTopology()
	if err != nil {
		return err
	}
	if err := c.channel.Qos(10, 0, false); err != nil {
		return fmt.Errorf("set RabbitMQ prefetch: %w", err)
	}
	deliveries, err := c.channel.ConsumeWithContext(ctx, queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume RabbitMQ queue: %w", err)
	}

	go c.consume(ctx, deliveries)
	go c.monitorConnection(ctx)
	c.log.Info("user integration-event consumer started", zap.String("queue", queueName))
	return nil
}

func (c *Consumer) Failures() <-chan error { return c.failures }

func (c *Consumer) monitorConnection(ctx context.Context) {
	notifications := c.connection.NotifyClose(make(chan *amqp091.Error, 1))
	select {
	case <-ctx.Done():
		return
	case reason, ok := <-notifications:
		var err error
		if ok && reason != nil {
			err = fmt.Errorf("RabbitMQ connection closed: %w", reason)
		} else {
			err = errors.New("RabbitMQ connection closed")
		}
		select {
		case c.failures <- err:
		default:
		}
	}
}

func (c *Consumer) declareTopology() (string, error) {
	exchanges := make(map[string]struct{}, len(bindings))
	for _, item := range bindings {
		if _, declared := exchanges[item.exchange]; declared {
			continue
		}
		if err := c.channel.ExchangeDeclare(item.exchange, "topic", true, false, false, false, nil); err != nil {
			return "", fmt.Errorf("declare exchange %q: %w", item.exchange, err)
		}
		exchanges[item.exchange] = struct{}{}
	}

	queue, err := c.channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return "", fmt.Errorf("declare queue %q: %w", queueName, err)
	}
	for _, item := range bindings {
		if err := c.channel.QueueBind(queue.Name, item.routingKey, item.exchange, false, nil); err != nil {
			return "", fmt.Errorf("bind routing key %q: %w", item.routingKey, err)
		}
	}
	return queue.Name, nil
}

func (c *Consumer) consume(ctx context.Context, deliveries <-chan amqp091.Delivery) {
	for delivery := range deliveries {
		err := c.dispatch(ctx, delivery.RoutingKey, delivery.Body)
		if err == nil {
			if ackErr := delivery.Ack(false); ackErr != nil {
				c.log.Error("ack integration event", zap.Error(ackErr))
			}
			continue
		}

		requeue := !appevents.IsPermanent(err) && ctx.Err() == nil
		c.log.Error("handle integration event",
			zap.String("routing_key", delivery.RoutingKey),
			zap.Bool("requeue", requeue),
			zap.Error(err),
		)
		if nackErr := delivery.Nack(false, requeue); nackErr != nil {
			c.log.Error("nack integration event", zap.Error(nackErr))
		}
	}
}

func (c *Consumer) dispatch(ctx context.Context, routingKey string, body []byte) error {
	switch routingKey {
	case "auth.user_registered":
		var payload struct {
			UserID   string `json:"UserID"`
			Username string `json:"Username"`
			Email    string `json:"Email"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return appevents.Permanent(fmt.Errorf("decode user registration: %w", err))
		}
		return c.handler.UserRegistered(ctx, appevents.UserRegistered(payload))

	case "matching.match_found", "matching.match_completed":
		var payload struct {
			MatchID string `json:"MatchID"`
			User1ID string `json:"User1ID"`
			User2ID string `json:"User2ID"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return appevents.Permanent(fmt.Errorf("decode matching event: %w", err))
		}
		userIDs := []string{payload.User1ID, payload.User2ID}
		if routingKey == "matching.match_found" {
			return c.handler.MatchFound(ctx, appevents.MatchFound{MatchID: payload.MatchID, UserIDs: userIDs})
		}
		return c.handler.MatchCompleted(ctx, appevents.MatchCompleted{MatchID: payload.MatchID, UserIDs: userIDs})
	default:
		return appevents.Permanent(fmt.Errorf("unsupported routing key %q", routingKey))
	}
}

func (c *Consumer) Close() error {
	var closeErr error
	c.closeOnce.Do(func() {
		if err := c.channel.Close(); err != nil && !errors.Is(err, amqp091.ErrClosed) {
			closeErr = err
		}
		if err := c.connection.Close(); err != nil && !errors.Is(err, amqp091.ErrClosed) && closeErr == nil {
			closeErr = err
		}
	})
	return closeErr
}

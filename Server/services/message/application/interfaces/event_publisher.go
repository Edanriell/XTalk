package interfaces

import (
	"context"

	"github.com/yourusername/connect/message-service/domain/events"
)

// EventPublisher defines the interface for publishing domain events
type EventPublisher interface {
	PublishMessageSent(ctx context.Context, event events.MessageSentEvent) error
	PublishMessageRead(ctx context.Context, event events.MessageReadEvent) error
	PublishMessageDeleted(ctx context.Context, event events.MessageDeletedEvent) error
}

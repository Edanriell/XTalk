package interfaces

import (
	"context"

	"github.com/yourusername/connect/matching-service/domain/events"
)

// EventPublisher defines the interface for publishing matching events
type EventPublisher interface {
	PublishUserJoinedQueue(ctx context.Context, event events.UserJoinedQueueEvent) error
	PublishUserLeftQueue(ctx context.Context, event events.UserLeftQueueEvent) error
	PublishMatchFound(ctx context.Context, event events.MatchFoundEvent) error
	PublishMatchCompleted(ctx context.Context, event events.MatchCompletedEvent) error
}

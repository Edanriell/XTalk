package leave_queue

import (
	repositories "XTalk/services/matching/domain/matches"
	"XTalk/services/matching/domain/matches/events"
	"context"
	"time"

	"go.uber.org/zap"
)

type Handler struct {
	queueRepo      repositories.MatchingQueueRepository
	eventPublisher interfaces.EventPublisher
	log            *zap.Logger
}

func NewHandler(
	queueRepo repositories.MatchingQueueRepository,
	eventPublisher interfaces.EventPublisher,
	log *zap.Logger,
) *Handler {
	return &Handler{
		queueRepo:      queueRepo,
		eventPublisher: eventPublisher,
		log:            log,
	}
}

func (h *Handler) Handle(ctx context.Context, cmd Command) (*Result, error) {
	// Remove from queue
	_, err := h.queueRepo.RemoveFromQueue(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}

	// Publish user left queue event
	go func() {
		pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer pubCancel()
		event := events.UserLeftQueueEvent{
			UserID:    cmd.UserID,
			Reason:    "cancelled",
			Timestamp: time.Now(),
		}
		if err := h.eventPublisher.PublishUserLeftQueue(pubCtx, event); err != nil {
			h.log.Error("failed to publish user_left_queue event", zap.String("user_id", cmd.UserID), zap.Error(err))
		}
	}()

	return &Result{
		UserID:  cmd.UserID,
		Success: true,
		Message: "Removed from matching queue",
	}, nil
}

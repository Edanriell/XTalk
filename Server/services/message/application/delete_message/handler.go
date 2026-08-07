package delete_message

import (
	"XTalk/services/message/application/interfaces"
	"XTalk/services/message/domain/entities"
	"XTalk/services/message/domain/events"
	"XTalk/services/message/domain/repositories"
	"context"
	"time"

	"go.uber.org/zap"
)

type Handler struct {
	repo           repositories.MessageRepository
	eventPublisher interfaces.EventPublisher
	log            *zap.Logger
}

func NewHandler(
	repo repositories.MessageRepository,
	eventPublisher interfaces.EventPublisher,
	log *zap.Logger,
) *Handler {
	return &Handler{
		repo:           repo,
		eventPublisher: eventPublisher,
		log:            log,
	}
}

func (h *Handler) Handle(ctx context.Context, cmd Command) (*Result, error) {
	// Retrieve message
	message, err := h.repo.FindByID(ctx, cmd.MessageID)
	if err != nil {
		return nil, err
	}

	// Check authorization (only sender can delete)
	if !message.IsSentBy(cmd.UserID) {
		return nil, entities.ErrUnauthorizedDelete
	}

	// Delete message
	if err := message.Delete(); err != nil {
		return nil, err
	}

	// Save the deleted message
	if err := h.repo.Save(ctx, message); err != nil {
		return nil, err
	}

	// Publish event (async communication via RabbitMQ)
	event := events.MessageDeletedEvent{
		MessageID: message.ID(),
		ChatID:    message.ChatID(),
		DeletedBy: cmd.UserID,
		Timestamp: time.Now(),
	}

	// Non-blocking event publish
	go func() {
		pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.eventPublisher.PublishMessageDeleted(pubCtx, event); err != nil {
			h.log.Error("failed to publish message_deleted event",
				zap.String("message_id", message.ID()), zap.Error(err))
		}
	}()

	return &Result{
		MessageID: message.ID(),
		DeletedAt: *message.DeletedAt(),
	}, nil
}

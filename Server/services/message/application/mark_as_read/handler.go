package mark_as_read

import (
	"XTalk/services/message/application/interfaces"
	"XTalk/services/message/domain/entities"
	"XTalk/services/message/domain/events"
	"XTalk/services/message/domain/repositories"
	"context"
	"time"

	"go.uber.org/zap"
)

// Handler handles the mark message as read command
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

	// Only the recipient (non-sender) can mark a message as read
	if message.SenderID() == cmd.UserID {
		return nil, entities.ErrUnauthorizedMarkRead
	}

	// Mark as read
	if err := message.MarkAsRead(); err != nil {
		return nil, err
	}

	// Save the updated message
	if err := h.repo.Save(ctx, message); err != nil {
		return nil, err
	}

	// Publish event (async communication via RabbitMQ)
	event := events.MessageReadEvent{
		MessageID: message.ID(),
		ChatID:    message.ChatID(),
		ReaderID:  cmd.UserID,
		Timestamp: time.Now(),
	}

	// Non-blocking event publish
	go func() {
		pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.eventPublisher.PublishMessageRead(pubCtx, event); err != nil {
			h.log.Error("failed to publish message_read event",
				zap.String("message_id", message.ID()), zap.Error(err))
		}
	}()

	return &Result{
		MessageID: message.ID(),
		ReadAt:    *message.ReadAt(),
	}, nil
}

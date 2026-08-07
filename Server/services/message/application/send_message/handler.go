package send_message

import (
	"XTalk/services/message/application/interfaces"
	"XTalk/services/message/domain/entities"
	"XTalk/services/message/domain/events"
	"XTalk/services/message/domain/repositories"
	"XTalk/services/message/domain/valueobjects"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

var (
	ErrContentTooLong = fmt.Errorf("message content too long (max %d chars)", entities.MaxContentLength)
	ErrContentEmpty   = errors.New("message content cannot be empty")
)

// Validate validates the command inputs at the application boundary.
func (c *Command) Validate() error {
	if strings.TrimSpace(c.Content) == "" {
		return ErrContentEmpty
	}
	if len(c.Content) > entities.MaxContentLength {
		return ErrContentTooLong
	}
	return nil
}

// Handler handles the send message command
type Handler struct {
	repo           repositories.MessageRepository
	chatValidator  interfaces.ChatValidator
	idGenerator    interfaces.IDGenerator
	eventPublisher interfaces.EventPublisher
	log            *zap.Logger
}

func NewHandler(
	repo repositories.MessageRepository,
	chatValidator interfaces.ChatValidator,
	idGenerator interfaces.IDGenerator,
	eventPublisher interfaces.EventPublisher,
	log *zap.Logger,
) *Handler {
	return &Handler{
		repo:           repo,
		chatValidator:  chatValidator,
		idGenerator:    idGenerator,
		eventPublisher: eventPublisher,
		log:            log,
	}
}

func (h *Handler) Handle(ctx context.Context, cmd Command) (*Result, error) {
	// Validate command inputs
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	// Validate chat exists
	exists, err := h.chatValidator.ChatExists(ctx, cmd.ChatID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, entities.ErrInvalidChatID
	}

	// Validate sender is participant
	isParticipant, err := h.chatValidator.IsParticipant(ctx, cmd.ChatID, cmd.SenderID)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, entities.ErrInvalidSenderID
	}

	// Validate message type
	msgType, err := valueobjects.NewMessageType(cmd.MessageType)
	if err != nil {
		return nil, err
	}

	// Create message entity
	messageID := h.idGenerator.Generate()
	message := entities.NewMessage(
		messageID,
		cmd.ChatID,
		cmd.SenderID,
		msgType,
		cmd.Content,
		cmd.Metadata,
	)

	// Validate message
	if err := message.Validate(); err != nil {
		return nil, err
	}

	// Save message
	if err := h.repo.Save(ctx, message); err != nil {
		return nil, err
	}

	// Publish event (async communication via RabbitMQ)
	event := events.MessageSentEvent{
		MessageID:   message.ID(),
		ChatID:      message.ChatID(),
		SenderID:    message.SenderID(),
		Content:     message.Content(),
		MessageType: message.MessageType().String(),
		Timestamp:   time.Now(),
	}

	// Publish event synchronously with a short timeout.
	pubCtx, pubCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pubCancel()
	if err := h.eventPublisher.PublishMessageSent(pubCtx, event); err != nil {
		h.log.Error("failed to publish message_sent event", zap.String("message_id", message.ID()), zap.Error(err))
	}

	return &Result{
		MessageID: message.ID(),
		CreatedAt: message.CreatedAt(),
	}, nil
}

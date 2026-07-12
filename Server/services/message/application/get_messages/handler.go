package get_messages

import (
	"XTalk/services/message/application/interfaces"
	repositories "XTalk/services/message/domain/messages"
	"context"
	"time"
)

// MessageDTO represents a message data transfer object
type MessageDTO struct {
	ID          string
	ChatID      string
	SenderID    string
	MessageType string
	Content     string
	Metadata    map[string]string
	IsRead      bool
	CreatedAt   time.Time
	ReadAt      *time.Time
	DeletedAt   *time.Time
}

// Result represents the result of getting messages

// Handler handles the get messages query
type Handler struct {
	repo          repositories.MessageRepository
	chatValidator interfaces.ChatValidator
}

func NewHandler(repo repositories.MessageRepository, chatValidator interfaces.ChatValidator) *Handler {
	return &Handler{repo: repo, chatValidator: chatValidator}
}

func (h *Handler) Handle(ctx context.Context, query Query) (*Result, error) {
	// Authorize: ensure caller is a participant
	if query.UserID != "" {
		ok, err := h.chatValidator.IsParticipant(ctx, query.ChatID, query.UserID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrNotChatParticipant
		}
	}

	// Set default limit if not provided
	if query.Limit == 0 {
		query.Limit = 50
	}

	// Retrieve messages
	messages, err := h.repo.FindByChatID(ctx, query.ChatID, query.Limit, query.Offset)
	if err != nil {
		return nil, err
	}

	// Convert to DTOs
	dtos := make([]MessageDTO, len(messages))
	for i, msg := range messages {
		dtos[i] = MessageDTO{
			ID:          msg.ID(),
			ChatID:      msg.ChatID(),
			SenderID:    msg.SenderID(),
			MessageType: msg.MessageType().String(),
			Content:     msg.Content(),
			Metadata:    msg.Metadata(),
			IsRead:      msg.IsRead(),
			CreatedAt:   msg.CreatedAt(),
			ReadAt:      msg.ReadAt(),
			DeletedAt:   msg.DeletedAt(),
		}
	}

	return &Result{Messages: dtos}, nil
}

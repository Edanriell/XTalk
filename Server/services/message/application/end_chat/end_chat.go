package end_chat

import (
	"context"
	"errors"

	"github.com/yourusername/connect/chat-service/domain/repositories"
)

// Command represents a request to end a chat
type Command struct {
	ChatID string
	UserID string // User requesting to end the chat
}

// Result represents the result of ending a chat
type Result struct {
	ChatID  string
	Success bool
	Message string
}

// ErrNotAuthorized is returned when user is not authorized
var ErrNotAuthorized = errors.New("not authorized to end this chat")

// Handler handles the end chat command
type Handler struct {
	chatRepo repositories.ChatRepository
}

func NewHandler(chatRepo repositories.ChatRepository) *Handler {
	return &Handler{chatRepo: chatRepo}
}

func (h *Handler) Handle(ctx context.Context, cmd Command) (*Result, error) {
	// Find chat
	chat, err := h.chatRepo.FindByID(ctx, cmd.ChatID)
	if err != nil {
		return nil, err
	}

	// Verify user is a participant
	if !chat.IsParticipant(cmd.UserID) {
		return nil, ErrNotAuthorized
	}

	// End chat
	if err := chat.End(); err != nil {
		return nil, err
	}

	// Save changes
	if err := h.chatRepo.Save(ctx, chat); err != nil {
		return nil, err
	}

	return &Result{
		ChatID:  chat.ID(),
		Success: true,
		Message: "Chat ended successfully",
	}, nil
}

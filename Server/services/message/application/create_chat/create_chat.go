package create_chat

import (
	"context"

	"github.com/yourusername/connect/chat-service/application/interfaces"
	"github.com/yourusername/connect/chat-service/domain/entities"
	"github.com/yourusername/connect/chat-service/domain/repositories"
)

// Command represents a request to create a new chat
type Command struct {
	Participant1 string
	Participant2 string
	MatchScore   float64
}

// Result represents the result of creating a chat
type Result struct {
	ChatID       string
	Participant1 string
	Participant2 string
	MatchScore   float64
	Success      bool
	Message      string
}

// Handler handles the create chat command
type Handler struct {
	chatRepo    repositories.ChatRepository
	idGenerator interfaces.IDGenerator
}

func NewHandler(
	chatRepo repositories.ChatRepository,
	idGenerator interfaces.IDGenerator,
) *Handler {
	return &Handler{
		chatRepo:    chatRepo,
		idGenerator: idGenerator,
	}
}

func (h *Handler) Handle(ctx context.Context, cmd Command) (*Result, error) {
	// Validate participants are different
	if cmd.Participant1 == cmd.Participant2 {
		return nil, entities.ErrCannotChatWithSelf
	}

	// Generate chat ID
	chatID := h.idGenerator.GenerateID()

	// Create new chat
	chat := entities.NewChat(chatID, cmd.Participant1, cmd.Participant2, cmd.MatchScore)

	// Save chat — relies on a DB-level unique constraint
	if err := h.chatRepo.Save(ctx, chat); err != nil {
		return nil, err
	}

	return &Result{
		ChatID:       chat.ID(),
		Participant1: chat.Participant1(),
		Participant2: chat.Participant2(),
		MatchScore:   chat.MatchScore(),
		Success:      true,
		Message:      "Chat created successfully",
	}, nil
}

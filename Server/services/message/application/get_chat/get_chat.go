package get_chat

import (
	"context"
	"errors"
	"time"

	"github.com/yourusername/connect/chat-service/domain/entities"
	"github.com/yourusername/connect/chat-service/domain/repositories"
)

// Query represents a request to get a chat by ID
type Query struct {
	ChatID string
	UserID string // User requesting the chat (to verify participation)
}

// DTO represents chat data transfer object
type DTO struct {
	ID           string     `json:"id"`
	Participant1 string     `json:"participant1"`
	Participant2 string     `json:"participant2"`
	Status       string     `json:"status"`
	MatchScore   float64    `json:"match_score"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
}

var ErrNotAuthorized = errors.New("not authorized to view this chat")

// Handler handles the get chat query
type Handler struct {
	chatRepo repositories.ChatRepository
}

func NewHandler(chatRepo repositories.ChatRepository) *Handler {
	return &Handler{chatRepo: chatRepo}
}

func (h *Handler) Handle(ctx context.Context, query Query) (*DTO, error) {
	chat, err := h.chatRepo.FindByID(ctx, query.ChatID)
	if err != nil {
		return nil, err
	}

	// Verify user is a participant
	if !chat.IsParticipant(query.UserID) {
		return nil, entities.ErrNotParticipant
	}

	dto := &DTO{
		ID:           chat.ID(),
		Participant1: chat.Participant1(),
		Participant2: chat.Participant2(),
		Status:       chat.Status().String(),
		MatchScore:   chat.MatchScore(),
		CreatedAt:    chat.CreatedAt(),
		UpdatedAt:    chat.UpdatedAt(),
	}

	if chat.EndedAt() != nil {
		endedAt := *chat.EndedAt()
		dto.EndedAt = &endedAt
	}

	return dto, nil
}

package get_user_chats

import (
	"context"
	"time"

	"github.com/yourusername/connect/chat-service/domain/repositories"
)

// Query represents a request to get all chats for a user
type Query struct {
	UserID string
	Limit  int
	Offset int
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

// DTOList represents a list of chat DTOs
type DTOList struct {
	Chats []*DTO `json:"chats"`
	Total int    `json:"total"`
}

// Handler handles the get user chats query
type Handler struct {
	chatRepo repositories.ChatRepository
}

func NewHandler(chatRepo repositories.ChatRepository) *Handler {
	return &Handler{chatRepo: chatRepo}
}

func (h *Handler) Handle(ctx context.Context, query Query) (*DTOList, error) {
	// Cap limit and offset to prevent abuse.
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 200 {
		query.Limit = 200
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	const maxOffset = 1_000_000
	if query.Offset > maxOffset {
		query.Offset = maxOffset
	}

	chats, err := h.chatRepo.FindByUser(ctx, query.UserID, query.Limit, query.Offset)
	if err != nil {
		return nil, err
	}

	dtos := make([]*DTO, 0, len(chats))
	for _, chat := range chats {
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

		dtos = append(dtos, dto)
	}

	total, err := h.chatRepo.CountByUser(ctx, query.UserID)
	if err != nil {
		return nil, err
	}

	return &DTOList{
		Chats: dtos,
		Total: total,
	}, nil
}

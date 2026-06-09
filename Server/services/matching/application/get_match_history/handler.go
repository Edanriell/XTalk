package get_match_history

import (
	repositories "XTalk/services/matching/domain/matches"
	"context"
	"time"
)

// MatchDTO represents a match data transfer object
type MatchDTO struct {
	MatchID     string
	MatchedWith string
	ChatID      string
	MatchScore  float64
	Status      string
	CreatedAt   time.Time
	CompletedAt *time.Time
}

// Handler handles the get match history query
type Handler struct {
	matchHistoryRepo repositories.MatchHistoryRepository
}

func NewHandler(matchHistoryRepo repositories.MatchHistoryRepository) *Handler {
	return &Handler{matchHistoryRepo: matchHistoryRepo}
}

func (h *Handler) Handle(ctx context.Context, query Query) (*Result, error) {
	// Set default limit if not provided
	if query.Limit == 0 {
		query.Limit = 20
	}
	if query.Limit > 200 {
		query.Limit = 200
	}
	const maxOffset = 1_000_000
	if query.Offset > maxOffset {
		query.Offset = maxOffset
	}

	// Retrieve match history
	matches, err := h.matchHistoryRepo.FindByUserID(ctx, query.UserID, query.Limit, query.Offset)
	if err != nil {
		return nil, err
	}

	// Convert to DTOs
	dtos := make([]MatchDTO, len(matches))
	for i, match := range matches {
		otherUserID, _ := match.GetOtherParticipant(query.UserID)

		dtos[i] = MatchDTO{
			MatchID:     match.ID(),
			MatchedWith: otherUserID,
			ChatID:      match.ChatID(),
			MatchScore:  match.MatchScore().Value(),
			Status:      match.Status(),
			CreatedAt:   match.CreatedAt(),
			CompletedAt: match.CompletedAt(),
		}
	}

	return &Result{Matches: dtos}, nil
}

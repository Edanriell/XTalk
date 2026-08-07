package get_matching_status

import (
	repositories "XTalk/services/matching/domain/repositories"
	"context"
)

type Handler struct {
	queueRepo        repositories.MatchingQueueRepository
	matchHistoryRepo repositories.MatchHistoryRepository
}

func NewHandler(
	queueRepo repositories.MatchingQueueRepository,
	matchHistoryRepo repositories.MatchHistoryRepository,
) *Handler {
	return &Handler{
		queueRepo:        queueRepo,
		matchHistoryRepo: matchHistoryRepo,
	}
}

func (h *Handler) Handle(ctx context.Context, query Query) (*Result, error) {
	// Check if in queue
	inQueue, err := h.queueRepo.IsInQueue(ctx, query.UserID)
	if err != nil {
		return nil, err
	}

	if inQueue {
		candidate, err := h.queueRepo.FindInQueue(ctx, query.UserID)
		if err != nil {
			return nil, err
		}

		return &Result{
			Status:   "in_queue",
			WaitTime: int(candidate.WaitTime().Seconds()),
			Priority: candidate.Priority(),
		}, nil
	}

	// Check for active match
	activeMatches, err := h.matchHistoryRepo.FindActiveByUserID(ctx, query.UserID)
	if err != nil {
		return nil, err
	}

	if len(activeMatches) > 0 {
		match := activeMatches[0]
		otherUserID, _ := match.GetOtherParticipant(query.UserID)

		return &Result{
			Status:      "matched",
			MatchID:     match.ID(),
			ChatID:      match.ChatID(),
			MatchedWith: otherUserID,
			MatchScore:  match.MatchScore().Value(),
		}, nil
	}

	return &Result{
		Status:  "idle",
		Message: "Not in queue and no active matches",
	}, nil
}

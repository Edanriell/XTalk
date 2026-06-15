package end_match

import (
	repositories "XTalk/services/matching/domain/matches"
	"XTalk/services/matching/domain/matches/events"
	"context"
	"time"

	"go.uber.org/zap"
)

// Handler handles the end match command
type Handler struct {
	matchHistoryRepo repositories.MatchHistoryRepository
	eventPublisher   interfaces.EventPublisher
	log              *zap.Logger
}

func NewHandler(
	matchHistoryRepo repositories.MatchHistoryRepository,
	eventPublisher interfaces.EventPublisher,
	log *zap.Logger,
) *Handler {
	return &Handler{
		matchHistoryRepo: matchHistoryRepo,
		eventPublisher:   eventPublisher,
		log:              log,
	}
}

func (h *Handler) Handle(ctx context.Context, cmd Command) (*Result, error) {
	// Retrieve match
	match, err := h.matchHistoryRepo.FindByID(ctx, cmd.MatchID)
	if err != nil {
		return nil, err
	}

	// Verify user is participant
	if !match.IsParticipant(cmd.UserID) {
		return nil, entities.ErrNotParticipant
	}

	// Complete match
	if err := match.Complete(); err != nil {
		return nil, err
	}

	// Save match
	if err := h.matchHistoryRepo.Save(ctx, match); err != nil {
		return nil, err
	}

	// Publish match completed event
	go func() {
		pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer pubCancel()
		duration := int64(0)
		if match.CompletedAt() != nil {
			duration = match.CompletedAt().Unix() - match.CreatedAt().Unix()
		}

		event := events.MatchCompletedEvent{
			MatchID:   match.ID(),
			User1ID:   match.User1ID(),
			User2ID:   match.User2ID(),
			Duration:  duration,
			Timestamp: time.Now(),
		}
		if err := h.eventPublisher.PublishMatchCompleted(pubCtx, event); err != nil {
			h.log.Error("failed to publish match_completed event", zap.String("match_id", match.ID()), zap.Error(err))
		}
	}()

	return &Result{
		MatchID:     match.ID(),
		Success:     true,
		Message:     "Match completed successfully",
		CompletedAt: *match.CompletedAt(),
	}, nil
}

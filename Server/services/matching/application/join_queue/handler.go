package join_queue

import (
	"XTalk/services/matching/application/interfaces"
	"XTalk/services/matching/application/services"
	"XTalk/services/matching/domain/entities"
	"XTalk/services/matching/domain/events"
	"XTalk/services/matching/domain/repositories"
	"XTalk/services/matching/domain/valueobjects"
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Validate validates the command inputs at the application boundary.
func (c *Command) Validate() error {
	if len(c.Interests) > 20 {
		return ErrTooManyInterests
	}
	for _, interest := range c.Interests {
		if len(interest) > 100 {
			return ErrInterestTooLong
		}
	}
	if c.Gender != "" {
		if _, ok := validGenders[c.Gender]; !ok {
			return ErrInvalidGender
		}
	}
	if len(c.Location) > 255 {
		return ErrLocationTooLong
	}
	return nil
}

// Result represents the result of joining the matching queue

// Handler handles the join queue command
type Handler struct {
	queueRepo        repositories.MatchingQueueRepository
	matchHistoryRepo repositories.MatchHistoryRepository
	userValidator    interfaces.UserValidator
	chatCreator      interfaces.ChatCreator
	idGenerator      interfaces.IDGenerator
	eventPublisher   interfaces.EventPublisher
	matchingAlgo     *services.MatchingAlgorithm
	log              *zap.Logger
}

func NewHandler(
	queueRepo repositories.MatchingQueueRepository,
	matchHistoryRepo repositories.MatchHistoryRepository,
	userValidator interfaces.UserValidator,
	chatCreator interfaces.ChatCreator,
	idGenerator interfaces.IDGenerator,
	eventPublisher interfaces.EventPublisher,
	matchingAlgo *services.MatchingAlgorithm,
	log *zap.Logger,
) *Handler {
	return &Handler{
		queueRepo:        queueRepo,
		matchHistoryRepo: matchHistoryRepo,
		userValidator:    userValidator,
		chatCreator:      chatCreator,
		idGenerator:      idGenerator,
		eventPublisher:   eventPublisher,
		matchingAlgo:     matchingAlgo,
		log:              log,
	}
}

func (h *Handler) Handle(ctx context.Context, cmd Command) (*Result, error) {
	// Validate command inputs
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	// Validate user exists
	exists, err := h.userValidator.UserExists(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, entities.ErrInvalidPreferences
	}

	// Check if already in queue
	inQueue, err := h.queueRepo.IsInQueue(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	if inQueue {
		return nil, entities.ErrAlreadyInQueue
	}

	// Create preferences
	ageRange, err := valueobjects.NewAgeRange(cmd.MinAge, cmd.MaxAge)
	if err != nil {
		return nil, err
	}

	preferences, err := valueobjects.NewPreferences(
		cmd.Age,
		ageRange,
		cmd.Interests,
		cmd.Gender,
		cmd.Location,
	)
	if err != nil {
		return nil, err
	}

	// Create matching candidate
	candidate := entities.NewMatchingCandidate(cmd.UserID, preferences)

	// Try to find a match immediately using pre-filtered candidates.
	compatibleCandidates, err := h.queueRepo.FindCompatibleCandidates(ctx, candidate, 100)
	if err != nil {
		return nil, err
	}

	matchedCandidate, matchScore, err := h.matchingAlgo.FindBestMatch(candidate, compatibleCandidates)

	if err == nil && matchedCandidate != nil {
		// Check if these users were recently matched; if so, try remaining candidates.
		hadRecent, recentErr := h.matchHistoryRepo.HasRecentMatch(ctx, candidate.UserID(), matchedCandidate.UserID())
		if recentErr != nil {
			h.log.Error("failed to check recent match", zap.Error(recentErr))
		}
		if hadRecent || recentErr != nil {
			filtered := make([]*entities.MatchingCandidate, 0, len(compatibleCandidates)-1)
			for _, c := range compatibleCandidates {
				if c.UserID() == matchedCandidate.UserID() {
					continue
				}
				recent, rErr := h.matchHistoryRepo.HasRecentMatch(ctx, candidate.UserID(), c.UserID())
				if rErr != nil {
					h.log.Error("failed to check recent match", zap.Error(rErr))
					continue
				}
				if !recent {
					filtered = append(filtered, c)
				}
			}
			matchedCandidate, matchScore, err = h.matchingAlgo.FindBestMatch(candidate, filtered)
		}
	}

	if err == nil && matchedCandidate != nil {
		matchID := h.idGenerator.Generate()
		match := entities.NewMatch(matchID, candidate.UserID(), matchedCandidate.UserID(), matchScore)

		// Create chat room
		chatID, err := h.chatCreator.CreateChat(ctx, candidate.UserID(), matchedCandidate.UserID(), matchScore.Value())
		if err != nil {
			return h.addToQueue(ctx, candidate)
		}

		// Assign chat to match
		if err := match.AssignChatRoom(chatID); err != nil {
			h.log.Error("failed to assign chat room to match",
				zap.String("match_id", matchID), zap.String("chat_id", chatID), zap.Error(err))
			return nil, err
		}

		// Save match
		if err := h.matchHistoryRepo.Save(ctx, match); err != nil {
			h.log.Error("match saved failed, orphaned chat may exist",
				zap.String("chat_id", chatID), zap.Error(err))
			return nil, err
		}

		// Atomically remove matched candidate from queue
		removed, err := h.queueRepo.RemoveFromQueue(ctx, matchedCandidate.UserID())
		if err != nil {
			h.log.Error("failed to remove matched candidate from queue",
				zap.String("user_id", matchedCandidate.UserID()), zap.Error(err))
			return nil, fmt.Errorf("failed to finalize match: %w", err)
		}
		if !removed {
			h.log.Info("candidate was already matched by another thread, queuing",
				zap.String("candidate_id", matchedCandidate.UserID()))
			return h.addToQueue(ctx, candidate)
		}

		// Publish match found event
		go func() {
			pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer pubCancel()
			event := events.MatchFoundEvent{
				MatchID:    match.ID(),
				User1ID:    match.User1ID(),
				User2ID:    match.User2ID(),
				MatchScore: matchScore.Value(),
				ChatID:     chatID,
				Timestamp:  time.Now(),
			}
			if err := h.eventPublisher.PublishMatchFound(pubCtx, event); err != nil {
				h.log.Error("failed to publish match_found event", zap.String("match_id", match.ID()), zap.Error(err))
			}
		}()

		return &Result{
			Status:        "matched",
			MatchID:       match.ID(),
			MatchedUserID: matchedCandidate.UserID(),
			ChatID:        chatID,
			MatchScore:    matchScore.Value(),
		}, nil
	}

	// No match found, add to queue
	return h.addToQueue(ctx, candidate)
}

func (h *Handler) addToQueue(ctx context.Context, candidate *entities.MatchingCandidate) (*Result, error) {
	if err := h.queueRepo.AddToQueue(ctx, candidate); err != nil {
		return nil, err
	}

	go func() {
		pubCtx, pubCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer pubCancel()
		event := events.UserJoinedQueueEvent{
			UserID:    candidate.UserID(),
			Timestamp: time.Now(),
		}
		if err := h.eventPublisher.PublishUserJoinedQueue(pubCtx, event); err != nil {
			h.log.Error("failed to publish user_joined_queue event", zap.String("user_id", candidate.UserID()), zap.Error(err))
		}
	}()

	return &Result{
		Status:  "queued",
		Message: "Added to matching queue, waiting for a match...",
	}, nil
}

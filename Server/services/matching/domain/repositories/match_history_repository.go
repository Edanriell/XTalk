package repositories

import (
	"context"

	"XTalk/services/matching/domain/entities"
)

// MatchHistoryRepository defines the interface for match history persistence
type MatchHistoryRepository interface {
	// Save saves a match
	Save(ctx context.Context, match *entities.Match) error

	// FindByID retrieves a match by ID
	FindByID(ctx context.Context, matchID string) (*entities.Match, error)

	// FindByUserID retrieves all matches for a user
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*entities.Match, error)

	// FindActiveByUserID retrieves active matches for a user
	FindActiveByUserID(ctx context.Context, userID string) ([]*entities.Match, error)

	// HasRecentMatch checks if two users have recently matched
	HasRecentMatch(ctx context.Context, user1ID, user2ID string) (bool, error)
}

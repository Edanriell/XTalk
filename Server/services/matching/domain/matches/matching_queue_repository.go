package repositories

import (
	"context"

	"github.com/yourusername/connect/matching-service/domain/entities"
)

// MatchingQueueRepository defines the interface for matching queue persistence
type MatchingQueueRepository interface {
	// AddToQueue adds a candidate to the matching queue
	AddToQueue(ctx context.Context, candidate *entities.MatchingCandidate) error

	// RemoveFromQueue atomically removes a candidate from the queue.
	// Returns true if a row was actually deleted, false if no row existed.
	RemoveFromQueue(ctx context.Context, userID string) (bool, error)

	// FindInQueue finds a candidate in the queue
	FindInQueue(ctx context.Context, userID string) (*entities.MatchingCandidate, error)

	// GetAllCandidates retrieves all candidates in the queue
	GetAllCandidates(ctx context.Context) ([]*entities.MatchingCandidate, error)

	// FindCompatibleCandidates retrieves pre-filtered candidates that pass
	// basic compatibility checks (age range, gender) at the database level.
	// This avoids loading all candidates into memory.
	FindCompatibleCandidates(ctx context.Context, candidate *entities.MatchingCandidate, limit int) ([]*entities.MatchingCandidate, error)

	// IsInQueue checks if a user is in the queue
	IsInQueue(ctx context.Context, userID string) (bool, error)

	// UpdatePriority updates a candidate's priority
	UpdatePriority(ctx context.Context, userID string, priority int) error

	// UpdateCandidate replaces the preferences for a candidate already in the queue
	UpdateCandidate(ctx context.Context, candidate *entities.MatchingCandidate) error
}

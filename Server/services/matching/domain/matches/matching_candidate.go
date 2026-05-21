package entities

import (
	"time"

	"github.com/yourusername/connect/matching-service/domain/valueobjects"
)

// MatchingCandidate represents a user in the matching queue
type MatchingCandidate struct {
	userID      string
	preferences valueobjects.Preferences
	joinedAt    time.Time
	priority    int
}

// NewMatchingCandidate creates a new matching candidate
func NewMatchingCandidate(userID string, preferences valueobjects.Preferences) *MatchingCandidate {
	return &MatchingCandidate{
		userID:      userID,
		preferences: preferences,
		joinedAt:    time.Now(),
		priority:    0,
	}
}

// ReconstructMatchingCandidate reconstructs a matching candidate from persistence
func ReconstructMatchingCandidate(
	userID string,
	preferences valueobjects.Preferences,
	joinedAt time.Time,
	priority int,
) *MatchingCandidate {
	return &MatchingCandidate{
		userID:      userID,
		preferences: preferences,
		joinedAt:    joinedAt,
		priority:    priority,
	}
}

// Getters
func (m *MatchingCandidate) UserID() string                        { return m.userID }
func (m *MatchingCandidate) Preferences() valueobjects.Preferences { return m.preferences }
func (m *MatchingCandidate) JoinedAt() time.Time                   { return m.joinedAt }
func (m *MatchingCandidate) Priority() int                         { return m.priority }

// Business methods

// IncreasePriority increases the candidate's priority (for longer wait times)
func (m *MatchingCandidate) IncreasePriority() {
	m.priority++
}

// WaitTime calculates how long the candidate has been waiting
func (m *MatchingCandidate) WaitTime() time.Duration {
	return time.Since(m.joinedAt)
}

// IsCompatibleWith checks basic compatibility with another candidate
func (m *MatchingCandidate) IsCompatibleWith(other *MatchingCandidate) bool {
	// Can't match with yourself
	if m.userID == other.userID {
		return false
	}

	// Check age range compatibility
	if !m.preferences.AgeRange().IsInRange(other.preferences.Age()) {
		return false
	}
	if !other.preferences.AgeRange().IsInRange(m.preferences.Age()) {
		return false
	}

	// Check gender preference compatibility (empty string = no preference)
	if m.preferences.Gender() != "" && m.preferences.Gender() != other.preferences.Gender() {
		return false
	}
	if other.preferences.Gender() != "" && other.preferences.Gender() != m.preferences.Gender() {
		return false
	}

	return true
}

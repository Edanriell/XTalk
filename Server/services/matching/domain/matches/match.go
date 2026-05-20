package entities

import (
	"time"

	"github.com/yourusername/connect/matching-service/domain/valueobjects"
)

// Match represents a successful match between two users
type Match struct {
	id          string
	user1ID     string
	user2ID     string
	matchScore  valueobjects.MatchScore
	chatID      string
	status      string // active, completed, cancelled
	createdAt   time.Time
	completedAt *time.Time
}

// NewMatch creates a new match
func NewMatch(id, user1ID, user2ID string, matchScore valueobjects.MatchScore) *Match {
	return &Match{
		id:         id,
		user1ID:    user1ID,
		user2ID:    user2ID,
		matchScore: matchScore,
		status:     "pending",
		createdAt:  time.Now(),
	}
}

// ReconstructMatch reconstructs a match from persistence
func ReconstructMatch(
	id string,
	user1ID string,
	user2ID string,
	matchScore valueobjects.MatchScore,
	chatID string,
	status string,
	createdAt time.Time,
	completedAt *time.Time,
) *Match {
	return &Match{
		id:          id,
		user1ID:     user1ID,
		user2ID:     user2ID,
		matchScore:  matchScore,
		chatID:      chatID,
		status:      status,
		createdAt:   createdAt,
		completedAt: completedAt,
	}
}

// Getters
func (m *Match) ID() string                          { return m.id }
func (m *Match) User1ID() string                     { return m.user1ID }
func (m *Match) User2ID() string                     { return m.user2ID }
func (m *Match) MatchScore() valueobjects.MatchScore { return m.matchScore }
func (m *Match) ChatID() string                      { return m.chatID }
func (m *Match) Status() string                      { return m.status }
func (m *Match) CreatedAt() time.Time                { return m.createdAt }
func (m *Match) CompletedAt() *time.Time             { return m.completedAt }

// Business methods

// AssignChatRoom assigns a chat room to the match
func (m *Match) AssignChatRoom(chatID string) error {
	if m.chatID != "" {
		return ErrChatAlreadyAssigned
	}
	m.chatID = chatID
	m.status = "active"
	return nil
}

// Complete marks the match as completed
func (m *Match) Complete() error {
	if m.status == "completed" {
		return ErrMatchAlreadyCompleted
	}
	now := time.Now()
	m.status = "completed"
	m.completedAt = &now
	return nil
}

// Cancel cancels the match
func (m *Match) Cancel() error {
	if m.status == "completed" {
		return ErrMatchAlreadyCompleted
	}
	m.status = "cancelled"
	return nil
}

// IsActive checks if the match is active
func (m *Match) IsActive() bool {
	return m.status == "active"
}

// IsParticipant checks if a user is a participant in this match
func (m *Match) IsParticipant(userID string) bool {
	return m.user1ID == userID || m.user2ID == userID
}

// GetOtherParticipant returns the other participant's ID
func (m *Match) GetOtherParticipant(userID string) (string, error) {
	if m.user1ID == userID {
		return m.user2ID, nil
	}
	if m.user2ID == userID {
		return m.user1ID, nil
	}
	return "", ErrNotParticipant
}

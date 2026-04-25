package chat

import (
	"time"

	"github.com/yourusername/connect/chat-service/domain/valueobjects"
)

// Chat represents a chat room between two participants
type Chat struct {
	id           string
	participant1 string
	participant2 string
	status       valueobjects.ChatStatus
	matchScore   float64
	createdAt    time.Time
	updatedAt    time.Time
	endedAt      *time.Time
}

// NewChat creates a new Chat entity
func NewChat(id, participant1, participant2 string, matchScore float64) *Chat {
	now := time.Now()
	return &Chat{
		id:           id,
		participant1: participant1,
		participant2: participant2,
		status:       valueobjects.ChatStatusActive,
		matchScore:   matchScore,
		createdAt:    now,
		updatedAt:    now,
	}
}

// ReconstructChat reconstructs a Chat entity from persistence
func ReconstructChat(
	id string,
	participant1 string,
	participant2 string,
	status valueobjects.ChatStatus,
	matchScore float64,
	createdAt time.Time,
	updatedAt time.Time,
	endedAt *time.Time,
) *Chat {
	return &Chat{
		id:           id,
		participant1: participant1,
		participant2: participant2,
		status:       status,
		matchScore:   matchScore,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
		endedAt:      endedAt,
	}
}

// Getters
func (c *Chat) ID() string                      { return c.id }
func (c *Chat) Participant1() string            { return c.participant1 }
func (c *Chat) Participant2() string            { return c.participant2 }
func (c *Chat) Status() valueobjects.ChatStatus { return c.status }
func (c *Chat) MatchScore() float64             { return c.matchScore }
func (c *Chat) CreatedAt() time.Time            { return c.createdAt }
func (c *Chat) UpdatedAt() time.Time            { return c.updatedAt }
func (c *Chat) EndedAt() *time.Time             { return c.endedAt }

// Business methods

// IsParticipant checks if a user is a participant in this chat
func (c *Chat) IsParticipant(userID string) bool {
	return c.participant1 == userID || c.participant2 == userID
}

// GetOtherParticipant returns the other participant's ID
func (c *Chat) GetOtherParticipant(userID string) (string, error) {
	if c.participant1 == userID {
		return c.participant2, nil
	}
	if c.participant2 == userID {
		return c.participant1, nil
	}
	return "", ErrNotParticipant
}

// End ends the chat
func (c *Chat) End() error {
	if !c.status.IsActive() {
		return ErrChatAlreadyEnded
	}

	now := time.Now()
	c.status = valueobjects.ChatStatusEnded
	c.endedAt = &now
	c.updatedAt = now
	return nil
}

// IsActive checks if chat is active
func (c *Chat) IsActive() bool {
	return c.status.IsActive()
}

// UpdateActivity updates the last activity timestamp
func (c *Chat) UpdateActivity() {
	c.updatedAt = time.Now()
}

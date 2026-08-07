package repositories

import (
	"context"

	"XTalk/services/chat/domain/entities"
)

// ChatRepository defines the interface for chat persistence
type ChatRepository interface {
	// Save creates or updates a chat
	Save(ctx context.Context, chat *entities.Chat) error

	// FindByID retrieves a chat by ID
	FindByID(ctx context.Context, id string) (*entities.Chat, error)

	// FindByParticipants retrieves a chat by participant IDs
	FindByParticipants(ctx context.Context, participant1, participant2 string) (*entities.Chat, error)

	// FindActiveByUser retrieves active chat for a user
	FindActiveByUser(ctx context.Context, userID string) (*entities.Chat, error)

	// FindByUser retrieves all chats for a user
	FindByUser(ctx context.Context, userID string, limit, offset int) ([]*entities.Chat, error)

	// CountByUser returns the total number of chats a user participates in.
	CountByUser(ctx context.Context, userID string) (int, error)

	// Delete removes a chat
	Delete(ctx context.Context, id string) error

	// ExistsActiveChat checks if user has an active chat
	ExistsActiveChat(ctx context.Context, userID string) (bool, error)
}

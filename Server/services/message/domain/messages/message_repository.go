package repositories

import (
	"context"

	"github.com/yourusername/connect/message-service/domain/entities"
)

// MessageRepository defines the interface for message persistence
type MessageRepository interface {
	// Save creates or updates a message
	Save(ctx context.Context, message *entities.Message) error

	// FindByID retrieves a message by ID
	FindByID(ctx context.Context, messageID string) (*entities.Message, error)

	// FindByChatID retrieves all messages for a chat room
	FindByChatID(ctx context.Context, chatID string, limit, offset int) ([]*entities.Message, error)

	// Delete removes a message (soft delete)
	Delete(ctx context.Context, messageID string) error

	// MarkAsRead marks a message as read
	MarkAsRead(ctx context.Context, messageID string) error

	// CountUnreadByChatID counts unread messages in a chat
	CountUnreadByChatID(ctx context.Context, chatID string, userID string) (int, error)

	// FindUnreadByChatID retrieves unread messages for a user in a chat
	FindUnreadByChatID(ctx context.Context, chatID string, userID string) ([]*entities.Message, error)
}

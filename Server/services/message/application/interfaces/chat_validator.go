package interfaces

import "context"

// ChatValidator defines the interface for validating chat rooms (dependency on chat-service)
type ChatValidator interface {
	// ChatExists checks if a chat room exists
	ChatExists(ctx context.Context, chatID string) (bool, error)

	// IsParticipant checks if a user is a participant in a chat
	IsParticipant(ctx context.Context, chatID string, userID string) (bool, error)
}

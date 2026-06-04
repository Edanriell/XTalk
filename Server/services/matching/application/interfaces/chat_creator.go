package interfaces

import "context"

// ChatCreator defines the interface for creating chat rooms (dependency on chat-service)
type ChatCreator interface {
	// CreateChat creates a new chat room between two users
	CreateChat(ctx context.Context, user1ID, user2ID string, matchScore float64) (chatID string, err error)
}

package interfaces

import "context"

// UserValidator defines the interface for validating users (dependency on user-service)
type UserValidator interface {
	// UserExists checks if a user exists
	UserExists(ctx context.Context, userID string) (bool, error)
}

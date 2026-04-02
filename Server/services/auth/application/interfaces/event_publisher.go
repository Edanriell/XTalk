package interfaces

import "context"

// UserRegisteredEvent is published when a new user registers.
type UserRegisteredEvent struct {
	UserID   string `json:"UserID"`
	Username string `json:"Username"`
	Email    string `json:"Email"`
}

// EventPublisher publishes domain events to a message broker.
type EventPublisher interface {
	PublishUserRegistered(ctx context.Context, event UserRegisteredEvent) error
	Close() error
}

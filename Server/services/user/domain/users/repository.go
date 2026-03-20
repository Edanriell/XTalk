package users

import "context"

// UserRepository is the persistence port required by the user application.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	Save(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email Email) (*User, error)
}

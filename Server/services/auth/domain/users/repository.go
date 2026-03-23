package users

import (
	"context"
)

// UserRepository is a domain interface (port)
type UserRepository interface {
	Save(ctx context.Context, user *entities.User) error
	FindByID(ctx context.Context, id string) (*entities.User, error)
	FindByEmail(ctx context.Context, email valueobjects.Email) (*entities.User, error)
	EmailExists(ctx context.Context, email valueobjects.Email) (bool, error)
	Delete(ctx context.Context, id string) error
}

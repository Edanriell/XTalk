package users

import "context"

type UserRepository interface {
	Save(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email Email) (*User, error)
	EmailExists(ctx context.Context, email Email) (bool, error)
	Delete(ctx context.Context, id string) error
}

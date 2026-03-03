package users

import "context"

type UserRepository interface {
	Save(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email Email) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	Delete(ctx context.Context, id string) error
	ExistsByEmail(ctx context.Context, email Email) (bool, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	FindActiveUsers(ctx context.Context, limit, offset int) ([]*User, error)
	UpdateStatus(ctx context.Context, id string, status Status) error
}

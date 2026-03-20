package create_user

import (
	"context"

	"XTalk/services/user/domain/users"
)

type Handler struct {
	userRepo users.UserRepository
}

func NewHandler(userRepo users.UserRepository) *Handler {
	return &Handler{userRepo: userRepo}
}

func (h *Handler) Handle(ctx context.Context, cmd Command) error {
	email, err := users.NewEmail(cmd.Email)
	if err != nil {
		return err
	}

	user, err := users.NewUser(cmd.UserID, cmd.Username, email)
	if err != nil {
		return err
	}
	return h.userRepo.Create(ctx, user)
}

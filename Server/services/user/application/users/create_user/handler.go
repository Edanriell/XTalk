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
	email, err := users.CreateEmail(cmd.Email)
	if err != nil {
		return err
	}

	user := users.CreateUser(cmd.UserID, cmd.Username, email, 0, "", "", "")
	return h.userRepo.Save(ctx, user)
}

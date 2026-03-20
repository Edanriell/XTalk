package delete_user

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
	if err := users.ValidateID(cmd.UserID); err != nil {
		return err
	}
	user, err := h.userRepo.FindByID(ctx, cmd.UserID)
	if err != nil {
		return err
	}

	user.Deactivate()

	if err := h.userRepo.Save(ctx, user); err != nil {
		return err
	}
	return nil
}

package update_status

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

func (h *Handler) Handle(ctx context.Context, cmd Command) (*Result, error) {
	status, err := users.NewStatus(cmd.Status)
	if err != nil {
		return nil, err
	}

	user, err := h.userRepo.FindByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive() {
		return nil, users.ErrUserInactive
	}

	user.UpdateStatus(status)

	if err := h.userRepo.Save(ctx, user); err != nil {
		return nil, err
	}

	return &Result{
		UserID:  user.ID(),
		Status:  status.String(),
		Success: true,
		Message: "Status updated successfully",
	}, nil
}

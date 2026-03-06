package get_user_by_email

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

func (h *Handler) Handle(ctx context.Context, query Query) (*Response, error) {
	email, err := users.CreateEmail(query.Email)
	if err != nil {
		return nil, err
	}

	user, err := h.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return &Response{
		ID:        user.ID(),
		Username:  user.Username(),
		Email:     user.Email().Value(),
		Age:       user.Age(),
		Gender:    user.Gender(),
		Country:   user.Country(),
		Language:  user.Language(),
		Interests: user.Interests(),
		Status:    user.Status().String(),
		Bio:       user.Bio(),
		AvatarURL: user.AvatarURL(),
		CreatedAt: user.CreatedAt(),
		UpdatedAt: user.UpdatedAt(),
		LastSeen:  user.LastSeen(),
		IsActive:  user.IsActive(),
	}, nil
}

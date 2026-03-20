package get_user_by_email

import (
	"context"

	"XTalk/services/user/application/users/readmodel"
	"XTalk/services/user/domain/users"
)

type Handler struct {
	userRepo users.UserRepository
}

func NewHandler(userRepo users.UserRepository) *Handler {
	return &Handler{userRepo: userRepo}
}

func (h *Handler) Handle(ctx context.Context, query Query) (*Response, error) {
	email, err := users.NewEmail(query.Email)
	if err != nil {
		return nil, err
	}

	user, err := h.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return readmodel.FromDomain(user), nil
}

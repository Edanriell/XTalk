package get_user

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
	if err := users.ValidateID(query.UserID); err != nil {
		return nil, err
	}
	user, err := h.userRepo.FindByID(ctx, query.UserID)
	if err != nil {
		return nil, err
	}

	return readmodel.FromDomain(user), nil
}

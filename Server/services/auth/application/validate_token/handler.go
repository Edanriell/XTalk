package validate_token

import (
	"XTalk/services/auth/application/interfaces"
	"context"
)

type Handler struct {
	tokenValidator interfaces.TokenValidator
	tokenBlacklist interfaces.TokenBlacklist
}

func NewHandler(
	tokenValidator interfaces.TokenValidator,
	tokenBlacklist interfaces.TokenBlacklist,
) *Handler {
	return &Handler{
		tokenValidator: tokenValidator,
		tokenBlacklist: tokenBlacklist,
	}
}

func (h *Handler) Handle(ctx context.Context, query Query) (*Response, error) {
	// Check if token is blacklisted
	if h.tokenBlacklist.IsBlacklisted(ctx, query.Token) {
		return &Response{Valid: false}, nil
	}

	// Validate token
	claims, err := h.tokenValidator.Validate(query.Token)
	if err != nil {
		return &Response{Valid: false}, nil
	}

	return &Response{
		Valid:  true,
		UserID: claims.UserID,
		Email:  claims.Email,
	}, nil
}

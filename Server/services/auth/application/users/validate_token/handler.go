package validate_token

import (
	"XTalk/services/auth/application/abstractions"
	"context"
)

type Handler struct {
	tokenValidator abstractions.TokenValidator
	tokenBlacklist abstractions.TokenBlacklist
}

func NewHandler(
	tokenValidator abstractions.TokenValidator,
	tokenBlacklist abstractions.TokenBlacklist,
) *Handler {
	return &Handler{
		tokenValidator: tokenValidator,
		tokenBlacklist: tokenBlacklist,
	}
}

func (h *Handler) Handle(ctx context.Context, query Query) (*Result, error) {
	// Check if token is blacklisted
	if h.tokenBlacklist.IsBlacklisted(ctx, query.Token) {
		return &Result{Valid: false}, nil
	}

	// Validate token
	claims, err := h.tokenValidator.Validate(query.Token)
	if err != nil {
		return &Result{Valid: false}, nil
	}

	return &Result{
		Valid:  true,
		UserID: claims.UserID,
		Email:  claims.Email,
	}, nil
}

// TODO
// Rename handler and refactor abstractions !

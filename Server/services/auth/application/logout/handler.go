package logout

import (
	"XTalk/services/auth/application/interfaces"
	"context"
	"time"
)

// Handler handles the logout command
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

func (h *Handler) Handle(ctx context.Context, cmd Command) (*Result, error) {
	// Validate the token to get its claims and ensure it's legitimate
	claims, err := h.tokenValidator.Validate(cmd.Token)
	if err != nil {
		// Token is already invalid, treat as successful logout
		return &Result{Success: true}, nil
	}

	// Verify the token belongs to the requesting user
	if claims.UserID != cmd.UserID {
		return &Result{Success: false}, nil
	}

	// Determine remaining token lifetime from claims.
	// If we can't determine it, fall back to 24h as a safe upper bound.
	expiry := 24 * time.Hour
	if claims.ExpiresAt != nil && !claims.ExpiresAt.IsZero() {
		remaining := time.Until(*claims.ExpiresAt)
		if remaining > 0 {
			expiry = remaining
		}
	}

	// Add the token to the blacklist so it cannot be used again.
	if err := h.tokenBlacklist.Add(ctx, cmd.Token, expiry); err != nil {
		return nil, err
	}

	return &Result{Success: true}, nil
}

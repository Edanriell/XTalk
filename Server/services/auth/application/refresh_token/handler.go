package refresh_token

import (
	"XTalk/services/auth/application/interfaces"
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrBlacklistedToken    = errors.New("refresh token has been revoked")
)

// Handler handles the refresh token command
type Handler struct {
	userRepo       repositories.UserRepository
	tokenValidator interfaces.TokenValidator
	tokenGenerator interfaces.TokenGenerator
	tokenBlacklist interfaces.TokenBlacklist
}

func NewHandler(
	userRepo repositories.UserRepository,
	tokenValidator interfaces.TokenValidator,
	tokenGenerator interfaces.TokenGenerator,
	tokenBlacklist interfaces.TokenBlacklist,
) *Handler {
	return &Handler{
		userRepo:       userRepo,
		tokenValidator: tokenValidator,
		tokenGenerator: tokenGenerator,
		tokenBlacklist: tokenBlacklist,
	}
}

func (h *Handler) Handle(ctx context.Context, cmd Command) (*Result, error) {
	// Check if the refresh token has been revoked
	if h.tokenBlacklist.IsBlacklisted(ctx, cmd.RefreshToken) {
		return nil, ErrBlacklistedToken
	}

	// Validate the refresh token and extract userID
	userID, expiresAt, err := h.tokenValidator.ValidateRefreshToken(cmd.RefreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Look up the user to get email for the new access token
	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Generate new token pair BEFORE blacklisting the old one.
	// This ensures the user is never locked out if token generation fails.
	accessToken, err := h.tokenGenerator.GenerateAccessToken(user.ID(), user.Email().String())
	if err != nil {
		return nil, err
	}

	refreshToken, err := h.tokenGenerator.GenerateRefreshToken(user.ID())
	if err != nil {
		return nil, err
	}

	// Now blacklist the old refresh token so it cannot be reused (rotation).
	// Use remaining token lifetime as TTL so the entry auto-expires from Redis.
	blacklistTTL := 7 * 24 * time.Hour // fallback: 7 days
	if expiresAt != nil {
		if remaining := time.Until(*expiresAt); remaining > 0 {
			blacklistTTL = remaining
		}
	}
	if err := h.tokenBlacklist.Add(ctx, cmd.RefreshToken, blacklistTTL); err != nil {
		return nil, err
	}

	return &Result{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

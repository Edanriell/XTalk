package login

import (
	"XTalk/services/auth/application/interfaces"
	"XTalk/services/auth/domain/users"
	"context"
	"crypto/subtle"
	"errors"

	"go.uber.org/zap"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrTooManyAttempts    = errors.New("too many failed login attempts")
)

// Handler handles the login command
type Handler struct {
	userRepo       users.UserRepository
	passwordHasher interfaces.PasswordHasher
	tokenGenerator interfaces.TokenGenerator
	rateLimiter    interfaces.RateLimiter
	log            *zap.Logger
}

func NewHandler(
	userRepo users.UserRepository,
	passwordHasher interfaces.PasswordHasher,
	tokenGenerator interfaces.TokenGenerator,
	rateLimiter interfaces.RateLimiter,
	log *zap.Logger,
) *Handler {
	return &Handler{
		userRepo:       userRepo,
		passwordHasher: passwordHasher,
		tokenGenerator: tokenGenerator,
		rateLimiter:    rateLimiter,
		log:            log,
	}
}

func (h *Handler) Handle(ctx context.Context, cmd Command) (*Result, error) {
	// Check rate limit
	if !h.rateLimiter.Allow(cmd.Email) {
		return nil, ErrTooManyAttempts
	}

	// Create email value object
	email, err := users.NewEmail(cmd.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Get user - using constant time to prevent timing attacks
	user, err := h.userRepo.FindByEmail(ctx, email)

	// Use dummy hash for timing attack prevention
	dummyHash := "$2a$12$dummy.hash.to.prevent.timing.attacks.and.user.enumeration"
	passwordHash := dummyHash
	userExists := false

	if err == nil && user != nil {
		passwordHash = user.PasswordHash()
		userExists = true
	}

	// Always check password (prevents timing attacks)
	passwordValid := h.passwordHasher.Compare(passwordHash, cmd.Password)

	// Constant-time comparison
	if subtle.ConstantTimeCompare([]byte{boolToByte(userExists && passwordValid)}, []byte{1}) != 1 {
		h.rateLimiter.IncrementFailure(cmd.Email)
		return nil, ErrInvalidCredentials
	}

	// Reset rate limit on success
	h.rateLimiter.Reset(cmd.Email)

	// Update last seen
	user.UpdateLastSeen()
	if err := h.userRepo.Save(ctx, user); err != nil {
		h.log.Warn("failed to update last_seen on login", zap.String("user_id", user.ID()), zap.Error(err))
	}

	// Generate tokens
	accessToken, err := h.tokenGenerator.GenerateAccessToken(user.ID(), email.String())
	if err != nil {
		return nil, err
	}

	refreshToken, err := h.tokenGenerator.GenerateRefreshToken(user.ID())
	if err != nil {
		return nil, err
	}

	return &Result{
		UserID:       user.ID(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

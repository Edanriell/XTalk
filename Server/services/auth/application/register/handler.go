package register

import (
	"XTalk/services/auth/application/interfaces"
	"XTalk/services/auth/domain/users"
	"context"

	"github.com/google/uuid"
)

// Handler handles the register command
type Handler struct {
	userRepo       Repository
	passwordHasher interfaces.PasswordHasher
	tokenGenerator interfaces.TokenGenerator
	validator      interfaces.Validator
}

// Repository keeps the credential aggregate and its integration event in one
// transaction. The event is relayed asynchronously by an outbound adapter.
type Repository interface {
	users.UserRepository
	CreateWithEvent(context.Context, *users.User, interfaces.UserRegisteredEvent) error
}

func NewHandler(
	userRepo Repository,
	passwordHasher interfaces.PasswordHasher,
	tokenGenerator interfaces.TokenGenerator,
	validator interfaces.Validator,
) *Handler {
	return &Handler{
		userRepo:       userRepo,
		passwordHasher: passwordHasher,
		tokenGenerator: tokenGenerator,
		validator:      validator,
	}
}

func (h *Handler) Handle(ctx context.Context, cmd Command) (*Result, error) {
	// Validate command
	if err := h.validator.ValidateUsername(cmd.Username); err != nil {
		return nil, err
	}

	if err := h.validator.ValidatePassword(cmd.Password); err != nil {
		return nil, err
	}

	// Create email value object
	email, err := users.NewEmail(cmd.Email)
	if err != nil {
		return nil, err
	}

	// Check if user exists
	exists, err := h.userRepo.EmailExists(ctx, email)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, users.ErrUserExists
	}

	// Hash password
	passwordHash, err := h.passwordHasher.Hash(cmd.Password)
	if err != nil {
		return nil, err
	}

	// Create user entity
	user := users.NewUser(
		uuid.New().String(),
		cmd.Username,
		email,
		passwordHash,
	)

	// Commit the credential aggregate and its integration event atomically.
	if err := h.userRepo.CreateWithEvent(ctx, user, interfaces.UserRegisteredEvent{
		UserID: user.ID(), Username: user.Username(), Email: user.Email().String(),
	}); err != nil {
		return nil, err
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

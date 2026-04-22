package register

import (
	"XTalk/services/auth/application/interfaces"
	"context"

	"github.com/google/uuid"
)

// Handler handles the register command
type Handler struct {
	userRepo       repositories.UserRepository
	passwordHasher interfaces.PasswordHasher
	tokenGenerator interfaces.TokenGenerator
	validator      interfaces.Validator
	eventPublisher interfaces.EventPublisher
}

func NewHandler(
	userRepo repositories.UserRepository,
	passwordHasher interfaces.PasswordHasher,
	tokenGenerator interfaces.TokenGenerator,
	validator interfaces.Validator,
	eventPublisher interfaces.EventPublisher,
) *Handler {
	return &Handler{
		userRepo:       userRepo,
		passwordHasher: passwordHasher,
		tokenGenerator: tokenGenerator,
		validator:      validator,
		eventPublisher: eventPublisher,
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
	email, err := valueobjects.NewEmail(cmd.Email)
	if err != nil {
		return nil, err
	}

	// Check if user exists
	exists, err := h.userRepo.EmailExists(ctx, email)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, entities.ErrUserExists
	}

	// Hash password
	passwordHash, err := h.passwordHasher.Hash(cmd.Password)
	if err != nil {
		return nil, err
	}

	// Create user entity
	user := entities.NewUser(
		uuid.New().String(),
		cmd.Username,
		email,
		passwordHash,
	)

	// Save user
	if err := h.userRepo.Save(ctx, user); err != nil {
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

	// Publish user registered event so other services can create the profile
	if h.eventPublisher != nil {
		_ = h.eventPublisher.PublishUserRegistered(ctx, interfaces.UserRegisteredEvent{
			UserID:   user.ID(),
			Username: cmd.Username,
			Email:    email.String(),
		})
	}

	return &Result{
		UserID:       user.ID(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

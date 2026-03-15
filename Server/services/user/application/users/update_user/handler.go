package update_user

import (
	"context"

	"XTalk/services/user/application/validation"
	"XTalk/services/user/domain/users"
)

type Handler struct {
	userRepo  users.UserRepository
	validator validation.Validator
}

func NewHandler(userRepo users.UserRepository, validator validation.Validator) *Handler {
	return &Handler{userRepo: userRepo, validator: validator}
}

func (h *Handler) Handle(ctx context.Context, cmd Command) (*Result, error) {
	if err := h.validator.ValidateUsername(cmd.Username); err != nil {
		return nil, err
	}
	if err := h.validator.ValidateAge(cmd.Age); err != nil {
		return nil, err
	}
	if err := h.validator.ValidateGender(cmd.Gender); err != nil {
		return nil, err
	}
	if err := h.validator.ValidateCountry(cmd.Country); err != nil {
		return nil, err
	}
	if err := h.validator.ValidateLanguage(cmd.Language); err != nil {
		return nil, err
	}
	if err := h.validator.ValidateInterests(cmd.Interests); err != nil {
		return nil, err
	}

	user, err := h.userRepo.FindByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive() {
		return nil, users.ErrUserInactive
	}

	if err := user.UpdateProfile(cmd.Username, cmd.Bio, cmd.Age, cmd.Gender, cmd.Country, cmd.Language); err != nil {
		return nil, err
	}
	if len(cmd.Interests) > 0 {
		if err := user.UpdateInterests(cmd.Interests); err != nil {
			return nil, err
		}
	}
	if cmd.AvatarURL != "" {
		user.UpdateAvatar(cmd.AvatarURL)
	}

	if err := h.userRepo.Save(ctx, user); err != nil {
		return nil, err
	}

	return &Result{UserID: user.ID(), Success: true, Message: "User updated successfully"}, nil
}

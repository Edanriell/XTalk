package update_user

import (
	"context"

	"XTalk/services/user/domain/users"
)

type Handler struct {
	userRepo users.UserRepository
}

func NewHandler(userRepo users.UserRepository) *Handler {
	return &Handler{userRepo: userRepo}
}

func (h *Handler) Handle(ctx context.Context, cmd Command) (*Result, error) {
	if err := users.ValidateID(cmd.UserID); err != nil {
		return nil, err
	}
	profile, err := users.NewProfile(users.ProfileInput{
		Username: cmd.Username, Bio: cmd.Bio, Age: cmd.Age, Gender: cmd.Gender,
		Country: cmd.Country, Language: cmd.Language, Interests: cmd.Interests,
		AvatarURL: cmd.AvatarURL,
	})
	if err != nil {
		return nil, err
	}

	user, err := h.userRepo.FindByID(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	if err := user.UpdateProfile(profile); err != nil {
		return nil, err
	}
	if err := h.userRepo.Save(ctx, user); err != nil {
		return nil, err
	}

	return &Result{UserID: user.ID()}, nil
}

package events

import (
	"context"
	"errors"
	"fmt"

	"XTalk/services/user/application/users/create_user"
	"XTalk/services/user/application/users/update_status"
	domainusers "XTalk/services/user/domain/users"
)

type UserRegistered struct {
	UserID   string
	Username string
	Email    string
}

type MatchFound struct {
	MatchID string
	UserIDs []string
}

type MatchCompleted struct {
	MatchID string
	UserIDs []string
}

type userCommands interface {
	Create(context.Context, create_user.Command) error
	UpdateStatus(context.Context, update_status.Command) (*update_status.Result, error)
}

// Handler translates integration events into application commands. Message
// brokers remain replaceable and contain no user-domain decisions.
type Handler struct {
	users userCommands
}

func NewHandler(users userCommands) *Handler {
	return &Handler{users: users}
}

func (h *Handler) UserRegistered(ctx context.Context, event UserRegistered) error {
	err := h.users.Create(ctx, create_user.Command{
		UserID: event.UserID, Username: event.Username, Email: event.Email,
	})
	if isPermanentDomainError(err) {
		return Permanent(err)
	}
	return err
}

func (h *Handler) MatchFound(ctx context.Context, event MatchFound) error {
	return h.updateStatuses(ctx, event.UserIDs, domainusers.StatusAway.String())
}

func (h *Handler) MatchCompleted(ctx context.Context, event MatchCompleted) error {
	return h.updateStatuses(ctx, event.UserIDs, domainusers.StatusOnline.String())
}

func (h *Handler) updateStatuses(ctx context.Context, userIDs []string, status string) error {
	if len(userIDs) == 0 {
		return Permanent(errors.New("matching event has no users"))
	}
	for _, userID := range userIDs {
		if err := domainusers.ValidateID(userID); err != nil {
			return Permanent(err)
		}
		if _, err := h.users.UpdateStatus(ctx, update_status.Command{UserID: userID, Status: status}); err != nil {
			if isPermanentDomainError(err) {
				return Permanent(err)
			}
			return fmt.Errorf("update user %q status: %w", userID, err)
		}
	}
	return nil
}

type permanentError struct{ error }

func (e permanentError) Unwrap() error { return e.error }

func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return permanentError{error: err}
}

func IsPermanent(err error) bool {
	var target permanentError
	return errors.As(err, &target)
}

func isPermanentDomainError(err error) bool {
	return errors.Is(err, domainusers.ErrUserAlreadyExists) ||
		errors.Is(err, domainusers.ErrUserInactive) ||
		domainusers.IsValidationError(err)
}

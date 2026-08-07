package events

import (
	"context"
	"errors"
	"testing"

	"XTalk/services/user/application/users/create_user"
	"XTalk/services/user/application/users/update_status"
	"XTalk/services/user/domain/users"
)

type fakeUserCommands struct {
	created   []create_user.Command
	statuses  []update_status.Command
	createErr error
	statusErr error
}

func (f *fakeUserCommands) Create(_ context.Context, command create_user.Command) error {
	f.created = append(f.created, command)
	return f.createErr
}

func (f *fakeUserCommands) UpdateStatus(_ context.Context, command update_status.Command) (*update_status.Result, error) {
	f.statuses = append(f.statuses, command)
	return &update_status.Result{}, f.statusErr
}

func TestHandlerTranslatesIntegrationEventsToCommands(t *testing.T) {
	commands := &fakeUserCommands{}
	handler := NewHandler(commands)

	if err := handler.UserRegistered(context.Background(), UserRegistered{
		UserID: "user-1", Username: "person_1", Email: "person@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if len(commands.created) != 1 || commands.created[0].UserID != "user-1" {
		t.Fatalf("created commands = %#v", commands.created)
	}

	if err := handler.MatchFound(context.Background(), MatchFound{UserIDs: []string{"user-1", "user-2"}}); err != nil {
		t.Fatal(err)
	}
	if len(commands.statuses) != 2 {
		t.Fatalf("status command count = %d, want 2", len(commands.statuses))
	}
	for _, command := range commands.statuses {
		if command.Status != users.StatusAway.String() {
			t.Fatalf("status = %q, want %q", command.Status, users.StatusAway)
		}
	}
}

func TestHandlerMarksInvalidEventsAsPermanent(t *testing.T) {
	commands := &fakeUserCommands{createErr: users.ErrEmailInvalid}
	err := NewHandler(commands).UserRegistered(context.Background(), UserRegistered{})
	if !IsPermanent(err) {
		t.Fatalf("IsPermanent(%domain) = false, want true", err)
	}
	if !errors.Is(err, users.ErrEmailInvalid) {
		t.Fatalf("wrapped error = %domain, want %domain", err, users.ErrEmailInvalid)
	}
}

func TestHandlerRejectsMatchingEventWithoutUsers(t *testing.T) {
	err := NewHandler(&fakeUserCommands{}).MatchFound(context.Background(), MatchFound{})
	if !IsPermanent(err) {
		t.Fatalf("IsPermanent(%domain) = false, want true", err)
	}
}

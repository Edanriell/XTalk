package register_test

import (
	"XTalk/services/auth/application/interfaces"
	"XTalk/services/auth/application/register"
	"XTalk/services/auth/domain/users"
	"context"
	"testing"
)

func TestHandlerStoresUserAndRegistrationEventAtomically(t *testing.T) {
	repo := &repositoryStub{}
	handler := register.NewHandler(repo, hasherStub{}, tokenGeneratorStub{}, validatorStub{})

	result, err := handler.Handle(context.Background(), register.Command{
		Username: "lauris", Email: "lauris@example.com", Password: "secret-password",
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	if repo.created == nil {
		t.Fatal("credential user was not stored")
	}
	if repo.event.UserID != result.UserID || repo.event.Email != "lauris@example.com" {
		t.Fatalf("stored event = %#v", repo.event)
	}
}

type repositoryStub struct {
	created *users.User
	event   interfaces.UserRegisteredEvent
}

func (r *repositoryStub) CreateWithEvent(_ context.Context, user *users.User, event interfaces.UserRegisteredEvent) error {
	r.created, r.event = user, event
	return nil
}
func (*repositoryStub) Save(context.Context, *users.User) error { return nil }
func (*repositoryStub) FindByID(context.Context, string) (*users.User, error) {
	return nil, users.ErrUserNotFound
}
func (*repositoryStub) FindByEmail(context.Context, users.Email) (*users.User, error) {
	return nil, users.ErrUserNotFound
}
func (*repositoryStub) EmailExists(context.Context, users.Email) (bool, error) { return false, nil }
func (*repositoryStub) Delete(context.Context, string) error                   { return nil }

type hasherStub struct{}

func (hasherStub) Hash(string) (string, error) { return "hash", nil }
func (hasherStub) Compare(string, string) bool { return true }

type tokenGeneratorStub struct{}

func (tokenGeneratorStub) GenerateAccessToken(string, string) (string, error) {
	return "access", nil
}
func (tokenGeneratorStub) GenerateRefreshToken(string) (string, error) {
	return "refresh", nil
}

type validatorStub struct{}

func (validatorStub) ValidateUsername(string) error { return nil }
func (validatorStub) ValidatePassword(string) error { return nil }

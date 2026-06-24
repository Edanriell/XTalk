package users_test

import (
	"errors"
	"testing"

	"XTalk/services/user/domain/users"
)

func TestNewEmailNormalizesAddress(t *testing.T) {
	email, err := users.NewEmail("  Person@Example.COM ")
	if err != nil {
		t.Fatalf("NewEmail() error = %domain", err)
	}
	if got, want := email.Value(), "person@example.com"; got != want {
		t.Fatalf("Email.Value() = %q, want %q", got, want)
	}
}

func TestNewUserRejectsZeroEmail(t *testing.T) {
	_, err := users.NewUser("user-1", "person_1", users.Email{})
	if !errors.Is(err, users.ErrEmailRequired) {
		t.Fatalf("NewUser() error = %domain, want %domain", err, users.ErrEmailRequired)
	}
}

func TestNewProfileRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		change func(*users.ProfileInput)
		want   error
	}{
		{name: "username", change: func(in *users.ProfileInput) { in.Username = "a b" }, want: users.ErrUsernameInvalid},
		{name: "age", change: func(in *users.ProfileInput) { in.Age = 12 }, want: users.ErrAgeInvalid},
		{name: "gender", change: func(in *users.ProfileInput) { in.Gender = "unknown" }, want: users.ErrGenderInvalid},
		{name: "country", change: func(in *users.ProfileInput) { in.Country = "Latvia" }, want: users.ErrCountryInvalid},
		{name: "language", change: func(in *users.ProfileInput) { in.Language = "eng" }, want: users.ErrLanguageInvalid},
		{name: "interest", change: func(in *users.ProfileInput) { in.Interests = []string{""} }, want: users.ErrInterestInvalid},
		{name: "avatar", change: func(in *users.ProfileInput) { in.AvatarURL = "file:///avatar.png" }, want: users.ErrAvatarURLInvalid},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validProfileInput()
			test.change(&input)
			_, err := users.NewProfile(input)
			if !errors.Is(err, test.want) {
				t.Fatalf("NewProfile() error = %domain, want %domain", err, test.want)
			}
		})
	}
}

func TestUpdateProfileIsAtomicAndDefensivelyCopiesInterests(t *testing.T) {
	user := newUser(t)
	input := validProfileInput()
	input.Interests = []string{" Go ", "go", "DDD"}
	profile, err := users.NewProfile(input)
	if err != nil {
		t.Fatalf("NewProfile() error = %domain", err)
	}
	input.Interests[0] = "changed"

	if err := user.UpdateProfile(profile); err != nil {
		t.Fatalf("UpdateProfile() error = %domain", err)
	}
	if got, want := user.Country(), "LV"; got != want {
		t.Fatalf("Country() = %q, want %q", got, want)
	}
	if got, want := user.Language(), "en"; got != want {
		t.Fatalf("Language() = %q, want %q", got, want)
	}
	interests := user.Interests()
	if len(interests) != 2 || interests[0] != "Go" || interests[1] != "DDD" {
		t.Fatalf("Interests() = %#domain, want [Go DDD]", interests)
	}
	interests[0] = "mutated"
	if user.Interests()[0] != "Go" {
		t.Fatal("Interests() exposed aggregate state")
	}
}

func TestInactiveUserCannotBeChanged(t *testing.T) {
	user := newUser(t)
	user.Deactivate()
	status, err := users.NewStatus("online")
	if err != nil {
		t.Fatal(err)
	}
	if err := user.UpdateStatus(status); !errors.Is(err, users.ErrUserInactive) {
		t.Fatalf("UpdateStatus() error = %domain, want %domain", err, users.ErrUserInactive)
	}
	profile, err := users.NewProfile(validProfileInput())
	if err != nil {
		t.Fatal(err)
	}
	if err := user.UpdateProfile(profile); !errors.Is(err, users.ErrUserInactive) {
		t.Fatalf("UpdateProfile() error = %domain, want %domain", err, users.ErrUserInactive)
	}
}

func TestUpdateStatusRejectsInvalidStatusValue(t *testing.T) {
	user := newUser(t)
	if err := user.UpdateStatus(users.Status("busy")); !errors.Is(err, users.ErrStatusInvalid) {
		t.Fatalf("UpdateStatus() error = %domain, want %domain", err, users.ErrStatusInvalid)
	}
}

func newUser(t *testing.T) *users.User {
	t.Helper()
	email, err := users.NewEmail("person@example.com")
	if err != nil {
		t.Fatal(err)
	}
	user, err := users.NewUser("user-1", "person_1", email)
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func validProfileInput() users.ProfileInput {
	return users.ProfileInput{
		Username: "person_1", Bio: " Hello ", Age: 25, Gender: " Other ",
		Country: "lv", Language: "EN", Interests: []string{"Go"},
		AvatarURL: "https://example.com/avatar.png",
	}
}

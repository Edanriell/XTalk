package users

import (
	"errors"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Email is a value object
type Email struct {
	value string
}

func NewEmail(value string) (Email, error) {
	value = strings.ToLower(strings.TrimSpace(value))

	if value == "" {
		return Email{}, errors.New("email cannot be empty")
	}

	if len(value) > 254 {
		return Email{}, errors.New("email too long")
	}

	if !emailRegex.MatchString(value) {
		return Email{}, errors.New("invalid email format")
	}

	return Email{value: value}, nil
}

func (e Email) String() string {
	return e.value
}

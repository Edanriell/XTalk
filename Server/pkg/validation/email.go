package validation

import (
	"errors"
	"regexp"
	"strings"
)

// TODO
// Move or remove ?

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

var (
	ErrEmailEmpty         = errors.New("email cannot be empty")
	ErrEmailTooLong       = errors.New("email too long")
	ErrEmailInvalidFormat = errors.New("invalid email format")
)

func ValidateEmail(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))

	if value == "" {
		return "", ErrEmailEmpty
	}
	if len(value) > 254 {
		return "", ErrEmailTooLong
	}

	if !emailRegex.MatchString(value) {
		return "", ErrEmailInvalidFormat
	}

	return value, nil
}

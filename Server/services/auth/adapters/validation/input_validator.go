package validation

import (
	"XTalk/services/auth/application/interfaces"
	"errors"
	"regexp"
	"unicode"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,30}$`)

// InputValidator implements interfaces.Validator
type InputValidator struct{}

func NewInputValidator() interfaces.Validator {
	return &InputValidator{}
}

func (v *InputValidator) ValidateUsername(username string) error {
	if username == "" {
		return errors.New("username is required")
	}

	if len(username) < 3 {
		return errors.New("username must be at least 3 characters")
	}

	if len(username) > 30 {
		return errors.New("username must be at most 30 characters")
	}

	if !usernameRegex.MatchString(username) {
		return errors.New("username can only contain letters, numbers, underscores and hyphens")
	}

	return nil
}

func (v *InputValidator) ValidatePassword(password string) error {
	if password == "" {
		return errors.New("password is required")
	}

	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	if len(password) > 72 {
		return errors.New("password must be at most 72 characters long")
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return errors.New("password must contain at least one number")
	}
	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}

	return nil
}

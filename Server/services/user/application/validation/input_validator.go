package validation

import (
	"errors"
	"regexp"
	"strings"
)

// TODO
// All this shit must be put in value objects
// Create and extend errors in domain users
// Create missing value objects !

// What should we do with messiging and grpc ? Should it be in infrastructure

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,30}$`)

type InputValidator struct{}

func NewInputValidator() Validator {
	return &InputValidator{}
}

func (v *InputValidator) ValidateUsername(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("username cannot be empty")
	}
	if len(username) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	if len(username) > 30 {
		return errors.New("username must be at most 30 characters")
	}
	if !usernameRegex.MatchString(username) {
		return errors.New("username can only contain letters, numbers, and underscores")
	}
	return nil
}

func (v *InputValidator) ValidateAge(age int) error {
	if age < 13 {
		return errors.New("age must be at least 13")
	}
	if age > 120 {
		return errors.New("age must be at most 120")
	}
	return nil
}

func (v *InputValidator) ValidateGender(gender string) error {
	gender = strings.ToLower(strings.TrimSpace(gender))
	valid := map[string]bool{"male": true, "female": true, "other": true}
	if !valid[gender] {
		return errors.New("gender must be male, female, or other")
	}
	return nil
}

func (v *InputValidator) ValidateCountry(country string) error {
	country = strings.TrimSpace(country)
	if country == "" {
		return nil
	}
	if len(country) != 2 {
		return errors.New("country code must be 2 characters (ISO 3166-1 alpha-2)")
	}
	return nil
}

func (v *InputValidator) ValidateLanguage(language string) error {
	language = strings.TrimSpace(language)
	if language == "" {
		return nil
	}
	if len(language) != 2 {
		return errors.New("language code must be 2 characters (ISO 639-1)")
	}
	return nil
}

func (v *InputValidator) ValidateInterests(interests []string) error {
	if len(interests) > 20 {
		return errors.New("maximum 20 interests allowed")
	}
	for _, interest := range interests {
		interest = strings.TrimSpace(interest)
		if interest == "" {
			return errors.New("interest cannot be empty")
		}
		if len(interest) > 50 {
			return errors.New("interest must be at most 50 characters")
		}
	}
	return nil
}

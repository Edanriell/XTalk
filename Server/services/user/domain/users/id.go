package users

import "strings"

func ValidateID(value string) error {
	if strings.TrimSpace(value) == "" {
		return ErrUserIDRequired
	}
	return nil
}

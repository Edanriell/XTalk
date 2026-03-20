package users

import (
	"regexp"
	"strings"
)

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`)

// Email is an immutable, normalized email address value object.
type Email struct {
	value string
}

func NewEmail(raw string) (Email, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case value == "":
		return Email{}, ErrEmailRequired
	case len(value) > 254:
		return Email{}, ErrEmailTooLong
	case !emailPattern.MatchString(value):
		return Email{}, ErrEmailInvalid
	default:
		return Email{value: value}, nil
	}
}

func (e Email) Value() string  { return e.value }
func (e Email) String() string { return e.value }

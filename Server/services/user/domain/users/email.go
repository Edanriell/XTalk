package users

import "XTalk/pkg/validation"

type Email struct {
	value string
}

func CreateEmail(value string) (Email, error) {
	normalised, err := validation.ValidateEmail(value)
	if err != nil {
		return Email{}, err
	}
	return Email{value: normalised}, nil
}

func (e Email) Value() string  { return e.value }
func (e Email) String() string { return e.value }

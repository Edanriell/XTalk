package users

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserInactive      = errors.New("user is inactive")

	ErrUserIDRequired   = errors.New("user ID is required")
	ErrUsernameRequired = errors.New("username is required")
	ErrUsernameInvalid  = errors.New("username must be 3 to 30 characters and contain only letters, numbers, and underscores")
	ErrEmailRequired    = errors.New("email is required")
	ErrEmailTooLong     = errors.New("email exceeds maximum length of 254 characters")
	ErrEmailInvalid     = errors.New("email has an invalid format")
	ErrAgeInvalid       = errors.New("age must be between 13 and 120")
	ErrGenderInvalid    = errors.New("gender must be male, female, other, or prefer_not_to_say")
	ErrCountryInvalid   = errors.New("country must be an ISO 3166-1 alpha-2 code")
	ErrLanguageInvalid  = errors.New("language must be an ISO 639-1 code")
	ErrBioTooLong       = errors.New("bio exceeds maximum length of 2000 characters")
	ErrTooManyInterests = errors.New("too many interests (maximum 20)")
	ErrInterestInvalid  = errors.New("interests must be non-empty and at most 50 characters")
	ErrAvatarURLTooLong = errors.New("avatar URL exceeds maximum length of 2048 characters")
	ErrAvatarURLInvalid = errors.New("avatar URL must use HTTP or HTTPS")
	ErrStatusInvalid    = errors.New("status must be online, offline, or away")
)

func IsValidationError(err error) bool {
	targets := [...]error{
		ErrUserIDRequired, ErrUsernameRequired, ErrUsernameInvalid,
		ErrEmailRequired, ErrEmailTooLong, ErrEmailInvalid,
		ErrAgeInvalid, ErrGenderInvalid, ErrCountryInvalid, ErrLanguageInvalid,
		ErrBioTooLong, ErrTooManyInterests, ErrInterestInvalid,
		ErrAvatarURLTooLong, ErrAvatarURLInvalid, ErrStatusInvalid,
	}
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

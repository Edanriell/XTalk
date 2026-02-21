package users

import "errors"

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUserInactive     = errors.New("user is inactive")
	ErrBioTooLong       = errors.New("bio exceeds maximum length of 2000 characters")
	ErrTooManyInterests = errors.New("too many interests (max 30)")
)

package users

import (
	"time"
)

type User struct {
	id           string
	username     string
	email        valueobjects.Email
	passwordHash string
	createdAt    time.Time
	updatedAt    time.Time
	lastSeen     time.Time
}

func NewUser(id, username string, email valueobjects.Email, passwordHash string) *User {
	now := time.Now()
	return &User{
		id:           id,
		username:     username,
		email:        email,
		passwordHash: passwordHash,
		createdAt:    now,
		updatedAt:    now,
		lastSeen:     now,
	}
}

func ReconstructUser(id, username string, email valueobjects.Email, passwordHash string, createdAt, updatedAt, lastSeen time.Time) *User {
	return &User{
		id:           id,
		username:     username,
		email:        email,
		passwordHash: passwordHash,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
		lastSeen:     lastSeen,
	}
}

func (u *User) ID() string                { return u.id }
func (u *User) Username() string          { return u.username }
func (u *User) Email() valueobjects.Email { return u.email }
func (u *User) PasswordHash() string      { return u.passwordHash }
func (u *User) CreatedAt() time.Time      { return u.createdAt }
func (u *User) UpdatedAt() time.Time      { return u.updatedAt }
func (u *User) LastSeen() time.Time       { return u.lastSeen }

func (u *User) UpdateLastSeen() {
	u.lastSeen = time.Now()
}

func (u *User) ChangeUsername(newUsername string) error {
	if newUsername == "" {
		return ErrInvalidUsername
	}
	u.username = newUsername
	u.updatedAt = time.Now()
	return nil
}

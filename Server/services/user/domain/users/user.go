package users

import "time"

// User is the aggregate root for the user domain.
type User struct {
	id        string
	username  string
	email     valueobjects.Email
	age       int
	gender    string
	country   string
	language  string
	interests []string
	status    valueobjects.Status
	bio       string
	avatarURL string
	createdAt time.Time
	updatedAt time.Time
	lastSeen  time.Time
	isActive  bool
}

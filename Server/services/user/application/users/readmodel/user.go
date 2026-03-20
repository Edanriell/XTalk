package readmodel

import (
	"time"

	"XTalk/services/user/domain/users"
)

type User struct {
	ID        string
	Username  string
	Email     string
	Age       int
	Gender    string
	Country   string
	Language  string
	Interests []string
	Status    string
	Bio       string
	AvatarURL string
	CreatedAt time.Time
	UpdatedAt time.Time
	LastSeen  time.Time
	IsActive  bool
}

func FromDomain(user *users.User) *User {
	return &User{
		ID: user.ID(), Username: user.Username(), Email: user.Email().Value(),
		Age: user.Age(), Gender: user.Gender(), Country: user.Country(),
		Language: user.Language(), Interests: user.Interests(), Status: user.Status().String(),
		Bio: user.Bio(), AvatarURL: user.AvatarURL(), CreatedAt: user.CreatedAt(),
		UpdatedAt: user.UpdatedAt(), LastSeen: user.LastSeen(), IsActive: user.IsActive(),
	}
}

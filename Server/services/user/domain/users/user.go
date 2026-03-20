package users

import (
	"strings"
	"time"
)

// User is the aggregate root for the user domain.
type User struct {
	id        string
	username  string
	email     Email
	age       int
	gender    string
	country   string
	language  string
	interests []string
	status    Status
	bio       string
	avatarURL string
	createdAt time.Time
	updatedAt time.Time
	lastSeen  time.Time
	isActive  bool
}

func NewUser(id, rawUsername string, email Email) (*User, error) {
	id = strings.TrimSpace(id)
	if err := ValidateID(id); err != nil {
		return nil, err
	}
	if email.Value() == "" {
		return nil, ErrEmailRequired
	}
	username, err := normalizeUsername(rawUsername)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &User{
		id: id, username: username, email: email, interests: []string{},
		status: StatusOffline, createdAt: now, updatedAt: now, lastSeen: now, isActive: true,
	}, nil
}

// ReconstructUser restores a previously persisted aggregate. It must only be
// called by trusted persistence adapters.
func ReconstructUser(
	id, username string,
	email Email,
	age int,
	gender, country, language string,
	interests []string,
	status Status,
	bio, avatarURL string,
	createdAt, updatedAt, lastSeen time.Time,
	isActive bool,
) *User {
	return &User{
		id: id, username: username, email: email, age: age, gender: gender,
		country: country, language: language, interests: cloneStrings(interests),
		status: status, bio: bio, avatarURL: avatarURL, createdAt: createdAt,
		updatedAt: updatedAt, lastSeen: lastSeen, isActive: isActive,
	}
}

func (u *User) ID() string           { return u.id }
func (u *User) Username() string     { return u.username }
func (u *User) Email() Email         { return u.email }
func (u *User) Age() int             { return u.age }
func (u *User) Gender() string       { return u.gender }
func (u *User) Country() string      { return u.country }
func (u *User) Language() string     { return u.language }
func (u *User) Interests() []string  { return cloneStrings(u.interests) }
func (u *User) Status() Status       { return u.status }
func (u *User) Bio() string          { return u.bio }
func (u *User) AvatarURL() string    { return u.avatarURL }
func (u *User) CreatedAt() time.Time { return u.createdAt }
func (u *User) UpdatedAt() time.Time { return u.updatedAt }
func (u *User) LastSeen() time.Time  { return u.lastSeen }
func (u *User) IsActive() bool       { return u.isActive }

func (u *User) UpdateProfile(profile Profile) error {
	if !u.isActive {
		return ErrUserInactive
	}
	u.username = profile.username
	u.bio = profile.bio
	u.age = profile.age
	u.gender = profile.gender
	u.country = profile.country
	u.language = profile.language
	u.interests = cloneStrings(profile.interests)
	u.avatarURL = profile.avatarURL
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) UpdateStatus(status Status) error {
	if !u.isActive {
		return ErrUserInactive
	}
	status, err := NewStatus(status.String())
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	u.status = status
	u.lastSeen = now
	u.updatedAt = now
	return nil
}

func (u *User) Deactivate() {
	if !u.isActive {
		return
	}
	u.isActive = false
	u.status = StatusOffline
	u.updatedAt = time.Now().UTC()
}

func cloneStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string(nil), values...)
}

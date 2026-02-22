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

func CreateUser(
	id,
	username string,
	email valueobjects.Email,
	age int,
	gender,
	country,
	language string,
) *User {
	return &User{
		id:        id,
		username:  username,
		email:     email,
		age:       age,
		gender:    gender,
		country:   country,
		language:  language,
		interests: []string{},
		status:    valueobjects.StatusOffline,
		createdAt: time.Now(),
		updatedAt: time.Now(),
		lastSeen:  time.Now(),
		isActive:  true,
	}
}

func ReconstructUser(
	id,
	username string,
	email valueobjects.Email,
	age int,
	gender,
	country,
	language string,
	interests []string,
	status valueobjects.Status,
	bio,
	avatarURL string,
	createdAt,
	updatedAt,
	lastSeen time.Time,
	isActive bool,
) *User {
	return &User{
		id:        id,
		username:  username,
		email:     email,
		age:       age,
		gender:    gender,
		country:   country,
		language:  language,
		interests: interests,
		status:    status,
		bio:       bio,
		avatarURL: avatarURL,
		createdAt: createdAt,
		updatedAt: updatedAt,
		lastSeen:  lastSeen,
		isActive:  isActive,
	}
}

func (u *User) ID() string                  { return u.id }
func (u *User) Username() string            { return u.username }
func (u *User) Email() valueobjects.Email   { return u.email }
func (u *User) Age() int                    { return u.age }
func (u *User) Gender() string              { return u.gender }
func (u *User) Country() string             { return u.country }
func (u *User) Language() string            { return u.language }
func (u *User) Interests() []string         { return u.interests }
func (u *User) Status() valueobjects.Status { return u.status }
func (u *User) Bio() string                 { return u.bio }
func (u *User) AvatarURL() string           { return u.avatarURL }
func (u *User) CreatedAt() time.Time        { return u.createdAt }
func (u *User) UpdatedAt() time.Time        { return u.updatedAt }
func (u *User) LastSeen() time.Time         { return u.lastSeen }
func (u *User) IsActive() bool              { return u.isActive }

func (u *User) UpdateProfile(
	username,
	bio string,
	age int,
	gender,
	country,
	language string,
) error {
	if len(bio) > 2000 {
		return ErrBioTooLong
	}

	u.username = username
	u.bio = bio
	u.age = age
	u.gender = gender
	u.country = country
	u.language = language
	u.updatedAt = time.Now()

	return nil
}

func (u *User) UpdateInterests(interests []string) error {
	if len(interests) > 30 {
		return ErrTooManyInterests
	}

	u.interests = interests
	u.updatedAt = time.Now()

	return nil
}

func (u *User) UpdateStatus(status valueobjects.Status) {
	u.status = status
	u.lastSeen = time.Now()
	u.updatedAt = time.Now()
}

func (u *User) UpdateAvatar(avatarURL string) {
	u.avatarURL = avatarURL
	u.updatedAt = time.Now()
}

func (u *User) Deactivate() {
	u.isActive = false
	u.status = valueobjects.StatusOffline
	u.updatedAt = time.Now()
}

func (u *User) Activate() {
	u.isActive = true
	u.updatedAt = time.Now()
}

func (u *User) UpdateLastSeen() {
	u.lastSeen = time.Now()
	u.updatedAt = time.Now()
}

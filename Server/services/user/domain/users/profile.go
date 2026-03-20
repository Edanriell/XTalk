package users

import (
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	maxBioLength       = 2000
	maxInterests       = 20
	maxInterestLength  = 50
	maxAvatarURLLength = 2048
)

var (
	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,30}$`)
	alpha2Pattern   = regexp.MustCompile(`^[a-zA-Z]{2}$`)
)

// Profile is a validated, normalized value object used for atomic profile updates.
type Profile struct {
	username  string
	bio       string
	age       int
	gender    string
	country   string
	language  string
	interests []string
	avatarURL string
}

type ProfileInput struct {
	Username  string
	Bio       string
	Age       int
	Gender    string
	Country   string
	Language  string
	Interests []string
	AvatarURL string
}

func NewProfile(input ProfileInput) (Profile, error) {
	username, err := normalizeUsername(input.Username)
	if err != nil {
		return Profile{}, err
	}
	if input.Age < 13 || input.Age > 120 {
		return Profile{}, ErrAgeInvalid
	}

	gender := strings.ToLower(strings.TrimSpace(input.Gender))
	switch gender {
	case "male", "female", "other", "prefer_not_to_say":
	default:
		return Profile{}, ErrGenderInvalid
	}

	country := strings.ToUpper(strings.TrimSpace(input.Country))
	if country != "" && !alpha2Pattern.MatchString(country) {
		return Profile{}, ErrCountryInvalid
	}
	language := strings.ToLower(strings.TrimSpace(input.Language))
	if language != "" && !alpha2Pattern.MatchString(language) {
		return Profile{}, ErrLanguageInvalid
	}

	bio := strings.TrimSpace(input.Bio)
	if utf8.RuneCountInString(bio) > maxBioLength {
		return Profile{}, ErrBioTooLong
	}

	interests, err := normalizeInterests(input.Interests)
	if err != nil {
		return Profile{}, err
	}
	avatarURL, err := normalizeAvatarURL(input.AvatarURL)
	if err != nil {
		return Profile{}, err
	}

	return Profile{
		username: username, bio: bio, age: input.Age, gender: gender,
		country: country, language: language, interests: interests, avatarURL: avatarURL,
	}, nil
}

func normalizeUsername(raw string) (string, error) {
	username := strings.TrimSpace(raw)
	if username == "" {
		return "", ErrUsernameRequired
	}
	if !usernamePattern.MatchString(username) {
		return "", ErrUsernameInvalid
	}
	return username, nil
}

func normalizeInterests(raw []string) ([]string, error) {
	if len(raw) > maxInterests {
		return nil, ErrTooManyInterests
	}

	result := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		interest := strings.TrimSpace(item)
		if interest == "" || utf8.RuneCountInString(interest) > maxInterestLength {
			return nil, ErrInterestInvalid
		}
		key := strings.ToLower(interest)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, interest)
	}
	return result, nil
}

func normalizeAvatarURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if len(value) > maxAvatarURLLength {
		return "", ErrAvatarURLTooLong
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", ErrAvatarURLInvalid
	}
	return value, nil
}

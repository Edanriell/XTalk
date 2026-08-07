package valueobjects

import "errors"

var (
	ErrInvalidAge      = errors.New("invalid age")
	ErrInvalidAgeRange = errors.New("invalid age range")
)

// Preferences represents user matching preferences
type Preferences struct {
	age       int
	ageRange  AgeRange
	interests []string
	gender    string
	location  string
}

// NewPreferences creates new matching preferences
func NewPreferences(age int, ageRange AgeRange, interests []string, gender, location string) (Preferences, error) {
	if age < 13 || age > 120 {
		return Preferences{}, ErrInvalidAge
	}

	return Preferences{
		age:       age,
		ageRange:  ageRange,
		interests: interests,
		gender:    gender,
		location:  location,
	}, nil
}

// Getters
func (p Preferences) Age() int            { return p.age }
func (p Preferences) AgeRange() AgeRange  { return p.ageRange }
func (p Preferences) Interests() []string { return p.interests }
func (p Preferences) Gender() string      { return p.gender }
func (p Preferences) Location() string    { return p.location }

// HasCommonInterests checks if there are common interests
func (p Preferences) HasCommonInterests(other Preferences) bool {
	set := make(map[string]struct{}, len(p.interests))
	for _, interest := range p.interests {
		set[interest] = struct{}{}
	}
	for _, otherInterest := range other.interests {
		if _, ok := set[otherInterest]; ok {
			return true
		}
	}
	return false
}

// CommonInterestCount counts common interests
func (p Preferences) CommonInterestCount(other Preferences) int {
	set := make(map[string]struct{}, len(p.interests))
	for _, interest := range p.interests {
		set[interest] = struct{}{}
	}
	count := 0
	for _, otherInterest := range other.interests {
		if _, ok := set[otherInterest]; ok {
			count++
		}
	}
	return count
}

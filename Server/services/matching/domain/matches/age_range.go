package valueobjects

// AgeRange represents a range of acceptable ages
type AgeRange struct {
	min int
	max int
}

// NewAgeRange creates a new age range
func NewAgeRange(min, max int) (AgeRange, error) {
	if min < 13 || max > 120 || min > max {
		return AgeRange{}, ErrInvalidAgeRange
	}

	return AgeRange{
		min: min,
		max: max,
	}, nil
}

// Getters
func (ar AgeRange) Min() int { return ar.min }
func (ar AgeRange) Max() int { return ar.max }

// IsInRange checks if an age is within the range
func (ar AgeRange) IsInRange(age int) bool {
	return age >= ar.min && age <= ar.max
}

// Overlaps checks if two age ranges overlap
func (ar AgeRange) Overlaps(other AgeRange) bool {
	return ar.min <= other.max && ar.max >= other.min
}

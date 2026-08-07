package valueobjects

import "errors"

var (
	ErrInvalidMatchScore = errors.New("match score must be between 0 and 1")
)

// MatchScore represents the compatibility score between two users
type MatchScore struct {
	value float64
}

// NewMatchScore creates a new match score
func NewMatchScore(value float64) (MatchScore, error) {
	if value < 0 || value > 1 {
		return MatchScore{}, ErrInvalidMatchScore
	}

	return MatchScore{value: value}, nil
}

// Value returns the score value
func (ms MatchScore) Value() float64 {
	return ms.value
}

// IsGoodMatch checks if the score indicates a good match (>= 0.7)
func (ms MatchScore) IsGoodMatch() bool {
	return ms.value >= 0.7
}

// IsExcellentMatch checks if the score indicates an excellent match (>= 0.9)
func (ms MatchScore) IsExcellentMatch() bool {
	return ms.value >= 0.9
}

// Percentage returns the score as a percentage
func (ms MatchScore) Percentage() int {
	return int(ms.value * 100)
}

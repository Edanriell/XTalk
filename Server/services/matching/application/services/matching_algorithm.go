package services

import (
	"math"

	"github.com/yourusername/connect/matching-service/domain/entities"
	"github.com/yourusername/connect/matching-service/domain/valueobjects"
)

// MatchingAlgorithm implements the matching logic (domain service)
type MatchingAlgorithm struct{}

// NewMatchingAlgorithm creates a new matching algorithm
func NewMatchingAlgorithm() *MatchingAlgorithm {
	return &MatchingAlgorithm{}
}

// FindBestMatch finds the best match for a candidate from a list of candidates
func (ma *MatchingAlgorithm) FindBestMatch(
	candidate *entities.MatchingCandidate,
	candidates []*entities.MatchingCandidate,
) (*entities.MatchingCandidate, valueobjects.MatchScore, error) {

	if len(candidates) == 0 {
		return nil, valueobjects.MatchScore{}, entities.ErrNoSuitableMatch
	}

	var bestCandidate *entities.MatchingCandidate
	var bestScore float64 = -1

	for _, other := range candidates {
		// Skip if same user
		if other.UserID() == candidate.UserID() {
			continue
		}

		// Check basic compatibility
		if !candidate.IsCompatibleWith(other) {
			continue
		}

		// Calculate match score
		score := ma.CalculateMatchScore(candidate, other)

		// Update best match if this score is higher
		if score > bestScore {
			bestScore = score
			bestCandidate = other
		}
	}

	if bestCandidate == nil {
		return nil, valueobjects.MatchScore{}, entities.ErrNoSuitableMatch
	}

	matchScore, err := valueobjects.NewMatchScore(bestScore)
	if err != nil {
		return nil, valueobjects.MatchScore{}, err
	}

	return bestCandidate, matchScore, nil
}

// CalculateMatchScore calculates compatibility score between two candidates
func (ma *MatchingAlgorithm) CalculateMatchScore(
	candidate1 *entities.MatchingCandidate,
	candidate2 *entities.MatchingCandidate,
) float64 {
	var score float64 = 0

	prefs1 := candidate1.Preferences()
	prefs2 := candidate2.Preferences()

	// 1. Age compatibility (30% weight)
	ageScore := ma.calculateAgeScore(prefs1, prefs2)
	score += ageScore * 0.3

	// 2. Common interests (40% weight)
	interestScore := ma.calculateInterestScore(prefs1, prefs2)
	score += interestScore * 0.4

	// 3. Wait time bonus (15% weight) - prioritize users who waited longer
	waitTimeScore := ma.calculateWaitTimeScore(candidate1, candidate2)
	score += waitTimeScore * 0.15

	// 4. Location proximity (15% weight)
	locationScore := ma.calculateLocationScore(prefs1, prefs2)
	score += locationScore * 0.15

	// Ensure score is between 0 and 1
	if score > 1 {
		score = 1
	}
	if score < 0 {
		score = 0
	}

	return score
}

// calculateAgeScore calculates age compatibility score
func (ma *MatchingAlgorithm) calculateAgeScore(prefs1, prefs2 valueobjects.Preferences) float64 {
	age1 := prefs1.Age()
	age2 := prefs2.Age()

	// Check if ages are within each other's preferred range
	inRange1 := prefs1.AgeRange().IsInRange(age2)
	inRange2 := prefs2.AgeRange().IsInRange(age1)

	if !inRange1 || !inRange2 {
		return 0
	}

	// Calculate how close the ages are
	ageDiff := math.Abs(float64(age1 - age2))

	// Perfect match if within 2 years
	if ageDiff <= 2 {
		return 1.0
	}

	// Decrease score as age difference increases
	score := 1.0 - (ageDiff / 20.0)
	if score < 0.5 {
		score = 0.5 // Minimum score if within range
	}

	return score
}

// calculateInterestScore calculates interest compatibility score
func (ma *MatchingAlgorithm) calculateInterestScore(prefs1, prefs2 valueobjects.Preferences) float64 {
	commonCount := prefs1.CommonInterestCount(prefs2)

	if commonCount == 0 {
		return 0.2 // Base score even with no common interests
	}

	// Max interests to consider is 5
	maxInterests := 5
	if commonCount > maxInterests {
		commonCount = maxInterests
	}

	return 0.2 + (float64(commonCount) / float64(maxInterests) * 0.8)
}

// calculateWaitTimeScore calculates wait time bonus score
func (ma *MatchingAlgorithm) calculateWaitTimeScore(candidate1, candidate2 *entities.MatchingCandidate) float64 {
	// Average wait time in minutes
	avgWaitMinutes := (candidate1.WaitTime().Minutes() + candidate2.WaitTime().Minutes()) / 2

	// More points for longer wait times
	if avgWaitMinutes < 1 {
		return 0.3
	} else if avgWaitMinutes < 5 {
		return 0.5
	} else if avgWaitMinutes < 15 {
		return 0.7
	} else {
		return 1.0
	}
}

// calculateLocationScore calculates location proximity score
func (ma *MatchingAlgorithm) calculateLocationScore(prefs1, prefs2 valueobjects.Preferences) float64 {
	loc1 := prefs1.Location()
	loc2 := prefs2.Location()

	// Simple exact match for now (can be enhanced with geolocation)
	if loc1 == loc2 {
		return 1.0
	}

	// Different locations get partial score
	return 0.5
}

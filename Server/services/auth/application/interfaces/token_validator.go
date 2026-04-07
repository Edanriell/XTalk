package interfaces

import "time"

// TokenClaims represents decoded token claims
type TokenClaims struct {
	UserID    string
	Email     string
	ExpiresAt *time.Time
}

// TokenValidator is an application port for token validation
type TokenValidator interface {
	Validate(token string) (*TokenClaims, error)
	ValidateRefreshToken(token string) (userID string, expiresAt *time.Time, err error)
}

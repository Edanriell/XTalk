package abstractions

// TokenClaims represents decoded token claims
type TokenClaims struct {
	UserID string
	Email  string
}

// TokenValidator is an application port for token validation
type TokenValidator interface {
	Validate(token string) (*TokenClaims, error)
}

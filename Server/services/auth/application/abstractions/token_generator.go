package abstractions

// TokenGenerator is an application port for token generation
type TokenGenerator interface {
	GenerateAccessToken(userID, email string) (string, error)
	GenerateRefreshToken(userID string) (string, error)
}

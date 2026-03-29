package security

import (
	"time"
)

// JWTTokenGenerator implements interfaces.TokenGenerator
type JWTTokenGenerator struct {
	secret        string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewJWTTokenGenerator(secret string, accessExpiry, refreshExpiry time.Duration) interfaces.TokenGenerator {
	return &JWTTokenGenerator{
		secret:        secret,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (g *JWTTokenGenerator) GenerateAccessToken(userID, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "connect-auth-service",
			Audience:  jwt.ClaimStrings{"connect-api"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(g.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(g.secret))
}

func (g *JWTTokenGenerator) GenerateRefreshToken(userID string) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    "connect-auth-service",
		Audience:  jwt.ClaimStrings{"connect-api"},
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(g.refreshExpiry)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(g.secret))
}

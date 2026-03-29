package security

import (
	"errors"
	"time"
)

// JWTTokenValidator implements interfaces.TokenValidator
type JWTTokenValidator struct {
	secret string
}

func NewJWTTokenValidator(secret string) interfaces.TokenValidator {
	return &JWTTokenValidator{secret: secret}
}

func (v *JWTTokenValidator) Validate(tokenString string) (*interfaces.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(v.secret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	tc := &interfaces.TokenClaims{
		UserID: claims.UserID,
		Email:  claims.Email,
	}
	if claims.ExpiresAt != nil {
		t := claims.ExpiresAt.Time
		tc.ExpiresAt = &t
	}

	return tc, nil
}

func (v *JWTTokenValidator) ValidateRefreshToken(tokenString string) (string, *time.Time, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(v.secret), nil
	})

	if err != nil {
		return "", nil, err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return "", nil, errors.New("invalid refresh token")
	}

	if claims.Subject == "" {
		return "", nil, errors.New("refresh token missing subject")
	}

	var expiresAt *time.Time
	if claims.ExpiresAt != nil {
		t := claims.ExpiresAt.Time
		expiresAt = &t
	}

	return claims.Subject, expiresAt, nil
}

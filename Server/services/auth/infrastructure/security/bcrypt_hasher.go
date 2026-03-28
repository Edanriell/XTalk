package security

import (
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// BcryptHasher implements interfaces.PasswordHasher
type BcryptHasher struct{}

func NewBcryptHasher() interfaces.PasswordHasher {
	return &BcryptHasher{}
}

func (h *BcryptHasher) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(bytes), err
}

func (h *BcryptHasher) Compare(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

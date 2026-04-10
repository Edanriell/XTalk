package abstractions

// PasswordHasher is an application port for password hashing
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hashedPassword, plainPassword string) bool
}

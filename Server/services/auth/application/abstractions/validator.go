package abstractions

// Validator is an application port for input validation
type Validator interface {
	ValidateUsername(username string) error
	ValidatePassword(password string) error
}

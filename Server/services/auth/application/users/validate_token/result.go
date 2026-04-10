package validate_token

type Result struct {
	Valid  bool
	UserID string
	Email  string
}

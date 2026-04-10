package validate_token

type Response struct {
	Valid  bool
	UserID string
	Email  string
}

package get_messages

type Query struct {
	ChatID string
	UserID string // caller — used for authorization
	Limit  int
	Offset int
}

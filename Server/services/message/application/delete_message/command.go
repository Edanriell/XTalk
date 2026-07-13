package delete_message

type Command struct {
	MessageID string
	UserID    string // For authorization
}

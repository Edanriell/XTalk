package get_user_chats

type Query struct {
	UserID string
	Limit  int
	Offset int
}

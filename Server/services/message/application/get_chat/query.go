package get_chat

type Query struct {
	ChatID string
	UserID string // User requesting the chat (to verify participation)
}

package send_message

type Command struct {
	ChatID      string
	SenderID    string
	MessageType string
	Content     string
	Metadata    map[string]string
}

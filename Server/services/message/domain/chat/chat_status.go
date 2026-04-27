package chat

import "errors"

// ChatStatus represents the status of a chat
type ChatStatus string

const (
	ChatStatusActive ChatStatus = "active"
	ChatStatusEnded  ChatStatus = "ended"
)

// NewChatStatus creates a new ChatStatus value object
func NewChatStatus(value string) (ChatStatus, error) {
	status := ChatStatus(value)

	switch status {
	case ChatStatusActive, ChatStatusEnded:
		return status, nil
	default:
		return "", errors.New("invalid chat status: must be active or ended")
	}
}

// String returns the string representation
func (s ChatStatus) String() string {
	return string(s)
}

// IsActive checks if status is active
func (s ChatStatus) IsActive() bool {
	return s == ChatStatusActive
}

// IsEnded checks if status is ended
func (s ChatStatus) IsEnded() bool {
	return s == ChatStatusEnded
}

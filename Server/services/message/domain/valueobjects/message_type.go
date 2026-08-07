package valueobjects

import "errors"

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeText  MessageType = "text"
	MessageTypeImage MessageType = "image"
	MessageTypeFile  MessageType = "file"
)

var (
	ErrInvalidMessageType = errors.New("invalid message type")
)

// NewMessageType creates and validates a MessageType
func NewMessageType(value string) (MessageType, error) {
	mt := MessageType(value)
	if !mt.IsValid() {
		return "", ErrInvalidMessageType
	}
	return mt, nil
}

// IsValid checks if the message type is valid
func (mt MessageType) IsValid() bool {
	switch mt {
	case MessageTypeText, MessageTypeImage, MessageTypeFile:
		return true
	default:
		return false
	}
}

// String returns the string representation
func (mt MessageType) String() string {
	return string(mt)
}

// IsText checks if message type is text
func (mt MessageType) IsText() bool {
	return mt == MessageTypeText
}

// IsImage checks if message type is image
func (mt MessageType) IsImage() bool {
	return mt == MessageTypeImage
}

// IsFile checks if message type is file
func (mt MessageType) IsFile() bool {
	return mt == MessageTypeFile
}

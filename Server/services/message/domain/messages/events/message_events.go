package events

import "time"

// MessageEvent represents a base message event for event-driven design
type MessageEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// MessageSentEvent represents when a message is sent
type MessageSentEvent struct {
	MessageID   string
	ChatID      string
	SenderID    string
	RecipientID string
	Content     string
	MessageType string
	Timestamp   time.Time
}

func (e MessageSentEvent) EventType() string     { return "message.sent" }
func (e MessageSentEvent) OccurredAt() time.Time { return e.Timestamp }

// MessageReadEvent represents when a message is read
type MessageReadEvent struct {
	MessageID string
	ChatID    string
	ReaderID  string
	Timestamp time.Time
}

func (e MessageReadEvent) EventType() string     { return "message.read" }
func (e MessageReadEvent) OccurredAt() time.Time { return e.Timestamp }

// MessageDeletedEvent represents when a message is deleted
type MessageDeletedEvent struct {
	MessageID string
	ChatID    string
	DeletedBy string
	Timestamp time.Time
}

func (e MessageDeletedEvent) EventType() string     { return "message.deleted" }
func (e MessageDeletedEvent) OccurredAt() time.Time { return e.Timestamp }

package entities

import (
	"time"

	"XTalk/services/message/domain/valueobjects"
)

// Message represents a chat message entity
type Message struct {
	id          string
	chatID      string
	senderID    string
	messageType valueobjects.MessageType
	content     string
	metadata    map[string]string
	isRead      bool
	createdAt   time.Time
	readAt      *time.Time
	deletedAt   *time.Time
}

// NewMessage creates a new Message entity
func NewMessage(id, chatID, senderID string, messageType valueobjects.MessageType, content string, metadata map[string]string) *Message {
	return &Message{
		id:          id,
		chatID:      chatID,
		senderID:    senderID,
		messageType: messageType,
		content:     content,
		metadata:    metadata,
		isRead:      false,
		createdAt:   time.Now(),
	}
}

// ReconstructMessage reconstructs a Message entity from persistence
func ReconstructMessage(
	id string,
	chatID string,
	senderID string,
	messageType valueobjects.MessageType,
	content string,
	metadata map[string]string,
	isRead bool,
	createdAt time.Time,
	readAt *time.Time,
	deletedAt *time.Time,
) *Message {
	return &Message{
		id:          id,
		chatID:      chatID,
		senderID:    senderID,
		messageType: messageType,
		content:     content,
		metadata:    metadata,
		isRead:      isRead,
		createdAt:   createdAt,
		readAt:      readAt,
		deletedAt:   deletedAt,
	}
}

// Getters
func (m *Message) ID() string                            { return m.id }
func (m *Message) ChatID() string                        { return m.chatID }
func (m *Message) SenderID() string                      { return m.senderID }
func (m *Message) MessageType() valueobjects.MessageType { return m.messageType }
func (m *Message) Content() string                       { return m.content }
func (m *Message) Metadata() map[string]string           { return m.metadata }
func (m *Message) IsRead() bool                          { return m.isRead }
func (m *Message) CreatedAt() time.Time                  { return m.createdAt }
func (m *Message) ReadAt() *time.Time                    { return m.readAt }
func (m *Message) DeletedAt() *time.Time                 { return m.deletedAt }

// Business methods

// MarkAsRead marks the message as read
func (m *Message) MarkAsRead() error {
	if m.isRead {
		return ErrMessageAlreadyRead
	}
	if m.IsDeleted() {
		return ErrMessageDeleted
	}

	now := time.Now()
	m.isRead = true
	m.readAt = &now
	return nil
}

// Delete soft deletes the message
func (m *Message) Delete() error {
	if m.IsDeleted() {
		return ErrMessageAlreadyDeleted
	}

	now := time.Now()
	m.deletedAt = &now
	return nil
}

// IsDeleted checks if the message is deleted
func (m *Message) IsDeleted() bool {
	return m.deletedAt != nil
}

// IsSentBy checks if the message was sent by the given user
func (m *Message) IsSentBy(userID string) bool {
	return m.senderID == userID
}

// Validate validates the message entity
func (m *Message) Validate() error {
	if m.id == "" {
		return ErrInvalidMessageID
	}
	if m.chatID == "" {
		return ErrInvalidChatID
	}
	if m.senderID == "" {
		return ErrInvalidSenderID
	}
	if m.content == "" && m.messageType == valueobjects.MessageTypeText {
		return ErrEmptyContent
	}
	if len(m.content) > MaxContentLength {
		return ErrContentTooLong
	}
	if len(m.metadata) > MaxMetadataEntries {
		return ErrMetadataTooLarge
	}
	return nil
}

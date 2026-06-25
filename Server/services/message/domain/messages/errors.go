package entities

import "errors"

const (
	// MaxContentLength is the maximum message content length (10 KB).
	MaxContentLength = 10_000
	// MaxMetadataEntries is the maximum number of metadata key-value pairs.
	MaxMetadataEntries = 20
)

var (
	// Message errors
	ErrInvalidMessageID      = errors.New("invalid message ID")
	ErrInvalidChatID         = errors.New("invalid chat ID")
	ErrInvalidSenderID       = errors.New("invalid sender ID")
	ErrEmptyContent          = errors.New("message content cannot be empty")
	ErrContentTooLong        = errors.New("message content exceeds maximum length")
	ErrMetadataTooLarge      = errors.New("too many metadata entries")
	ErrMessageAlreadyRead    = errors.New("message already marked as read")
	ErrMessageDeleted        = errors.New("message has been deleted")
	ErrMessageAlreadyDeleted = errors.New("message already deleted")
	ErrMessageNotFound       = errors.New("message not found")
	ErrUnauthorizedDelete    = errors.New("unauthorized to delete this message")
	ErrUnauthorizedMarkRead  = errors.New("only the recipient can mark a message as read")
)

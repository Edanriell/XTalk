package entities

import "errors"

var (
	// ErrChatNotFound is returned when a chat is not found
	ErrChatNotFound = errors.New("chat not found")
	// ErrChatAlreadyExists is returned when a chat already exists
	ErrChatAlreadyExists = errors.New("chat already exists")
	// ErrChatAlreadyEnded is returned when trying to perform operations on ended chat
	ErrChatAlreadyEnded = errors.New("chat already ended")
	// ErrNotParticipant is returned when user is not a participant
	ErrNotParticipant = errors.New("user is not a participant in this chat")
	// ErrCannotChatWithSelf is returned when user tries to chat with themselves
	ErrCannotChatWithSelf = errors.New("cannot create chat with yourself")
)

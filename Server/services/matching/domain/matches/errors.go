package entities

import "errors"

var (
	// Match errors
	ErrMatchNotFound         = errors.New("match not found")
	ErrChatAlreadyAssigned   = errors.New("chat room already assigned to match")
	ErrMatchAlreadyCompleted = errors.New("match already completed")
	ErrNotParticipant        = errors.New("user is not a participant in this match")

	// Matching candidate errors
	ErrCandidateNotFound  = errors.New("candidate not found in queue")
	ErrAlreadyInQueue     = errors.New("user already in matching queue")
	ErrNoSuitableMatch    = errors.New("no suitable match found")
	ErrInvalidPreferences = errors.New("invalid matching preferences")

)

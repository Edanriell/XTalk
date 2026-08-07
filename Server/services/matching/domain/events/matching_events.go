package events

import "time"

// MatchingEvent represents a base matching event for event-driven design
type MatchingEvent interface {
	EventType() string
	OccurredAt() time.Time
}

// UserJoinedQueueEvent represents when a user joins the matching queue
type UserJoinedQueueEvent struct {
	UserID    string
	Timestamp time.Time
}

func (e UserJoinedQueueEvent) EventType() string     { return "matching.user_joined_queue" }
func (e UserJoinedQueueEvent) OccurredAt() time.Time { return e.Timestamp }

// UserLeftQueueEvent represents when a user leaves the matching queue
type UserLeftQueueEvent struct {
	UserID    string
	Reason    string // "cancelled", "matched", "timeout"
	Timestamp time.Time
}

func (e UserLeftQueueEvent) EventType() string     { return "matching.user_left_queue" }
func (e UserLeftQueueEvent) OccurredAt() time.Time { return e.Timestamp }

// MatchFoundEvent represents when a match is found
type MatchFoundEvent struct {
	MatchID    string
	User1ID    string
	User2ID    string
	MatchScore float64
	ChatID     string
	Timestamp  time.Time
}

func (e MatchFoundEvent) EventType() string     { return "matching.match_found" }
func (e MatchFoundEvent) OccurredAt() time.Time { return e.Timestamp }

// MatchCompletedEvent represents when a match is completed
type MatchCompletedEvent struct {
	MatchID   string
	User1ID   string
	User2ID   string
	Duration  int64 // in seconds
	Timestamp time.Time
}

func (e MatchCompletedEvent) EventType() string     { return "matching.match_completed" }
func (e MatchCompletedEvent) OccurredAt() time.Time { return e.Timestamp }

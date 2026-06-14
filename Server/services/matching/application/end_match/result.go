package end_match

import "time"

type Result struct {
	MatchID     string
	Success     bool
	Message     string
	CompletedAt time.Time
}

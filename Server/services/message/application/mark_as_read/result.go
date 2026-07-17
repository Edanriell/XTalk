package mark_as_read

import "time"

type Result struct {
	MessageID string
	ReadAt    time.Time
}

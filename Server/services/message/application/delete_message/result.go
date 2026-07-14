package delete_message

import "time"

type Result struct {
	MessageID string
	DeletedAt time.Time
}

package users

import "strings"

type Status string

const (
	StatusOnline  Status = "online"
	StatusOffline Status = "offline"
	StatusAway    Status = "away"
)

func NewStatus(value string) (Status, error) {
	status := Status(strings.ToLower(strings.TrimSpace(value)))
	switch status {
	case StatusOnline, StatusOffline, StatusAway:
		return status, nil
	default:
		return "", ErrStatusInvalid
	}
}

func (s Status) String() string { return string(s) }

package users

import "errors"

type Status string

const (
	StatusOnline  Status = "online"
	StatusOffline Status = "offline"
	StatusAway    Status = "away"
)

func NewStatus(value string) (Status, error) {
	s := Status(value)
	switch s {
	case StatusOnline, StatusOffline, StatusAway:
		return s, nil
	default:
		return "", errors.New("invalid status: must be online, offline, or away")
	}
}

func (s Status) String() string { return string(s) }

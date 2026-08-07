package get_user_chats

import "time"

type Response struct {
	ID           string     `json:"id"`
	Participant1 string     `json:"participant1"`
	Participant2 string     `json:"participant2"`
	Status       string     `json:"status"`
	MatchScore   float64    `json:"match_score"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
}

type DTO = Response

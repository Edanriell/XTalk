package create_chat

type Result struct {
	ChatID       string
	Participant1 string
	Participant2 string
	MatchScore   float64
	Success      bool
	Message      string
}

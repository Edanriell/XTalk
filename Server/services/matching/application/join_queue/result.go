package join_queue

type Result struct {
	Status        string // "queued" or "matched"
	Message       string
	MatchID       string  // Only if matched
	MatchedUserID string  // Only if matched
	ChatID        string  // Only if matched
	MatchScore    float64 // Only if matched
}

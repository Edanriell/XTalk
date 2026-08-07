package get_matching_status

type Response struct {
	Status      string // "idle", "in_queue", "matched"
	Message     string
	WaitTime    int     // seconds (only if in_queue)
	Priority    int     // (only if in_queue)
	MatchID     string  // (only if matched)
	ChatID      string  // (only if matched)
	MatchedWith string  // (only if matched)
	MatchScore  float64 // (only if matched)
}

type Result = Response

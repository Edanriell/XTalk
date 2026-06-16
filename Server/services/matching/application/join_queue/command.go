package join_queue

type Command struct {
	UserID    string
	Age       int
	MinAge    int
	MaxAge    int
	Interests []string
	Gender    string
	Location  string
}

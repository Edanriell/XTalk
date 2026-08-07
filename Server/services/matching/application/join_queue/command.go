package join_queue

import "errors"

var (
	ErrTooManyInterests = errors.New("too many interests")
	ErrInterestTooLong  = errors.New("interest is too long")
	ErrInvalidGender    = errors.New("invalid gender")
	ErrLocationTooLong  = errors.New("location is too long")
)

var validGenders = map[string]struct{}{
	"male": {}, "female": {}, "non_binary": {}, "other": {},
}

type Command struct {
	UserID    string
	Age       int
	MinAge    int
	MaxAge    int
	Interests []string
	Gender    string
	Location  string
}

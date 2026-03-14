package update_user

type Command struct {
	UserID    string
	Username  string
	Bio       string
	Age       int
	Gender    string
	Country   string
	Language  string
	Interests []string
	AvatarURL string
}

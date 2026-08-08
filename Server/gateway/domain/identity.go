package domain

// Identity is the authenticated actor propagated through gateway use cases.
// It is transport-agnostic and contains no JWT or HTTP implementation detail.
type Identity struct {
	UserID string
	Email  string
}

func (i Identity) IsAuthenticated() bool { return i.UserID != "" }

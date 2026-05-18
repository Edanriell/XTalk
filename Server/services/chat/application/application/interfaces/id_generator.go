package interfaces

// IDGenerator generates unique IDs
type IDGenerator interface {
	GenerateID() string
}

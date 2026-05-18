package idgen

import (
	"github.com/google/uuid"
	"github.com/yourusername/connect/chat-service/application/interfaces"
)

// UUIDGenerator implements interfaces.IDGenerator using UUID
type UUIDGenerator struct{}

// NewUUIDGenerator creates a new UUIDGenerator
func NewUUIDGenerator() interfaces.IDGenerator {
	return &UUIDGenerator{}
}

// GenerateID generates a new UUID
func (g *UUIDGenerator) GenerateID() string {
	return uuid.New().String()
}

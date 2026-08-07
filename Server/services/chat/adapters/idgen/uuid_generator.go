package idgen

import (
	"XTalk/services/chat/application/interfaces"
	"github.com/google/uuid"
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

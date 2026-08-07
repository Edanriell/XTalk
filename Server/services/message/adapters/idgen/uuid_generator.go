package idgen

import (
	"XTalk/services/message/application/interfaces"
	"github.com/google/uuid"
)

// UUIDGenerator implements IDGenerator using UUID
type UUIDGenerator struct{}

// NewUUIDGenerator creates a new UUID generator
func NewUUIDGenerator() interfaces.IDGenerator {
	return &UUIDGenerator{}
}

// Generate generates a new UUID
func (g *UUIDGenerator) Generate() string {
	return uuid.New().String()
}

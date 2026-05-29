package idgen

import (
	"github.com/google/uuid"
	"github.com/yourusername/connect/matching-service/application/interfaces"
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

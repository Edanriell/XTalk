package requestid

import (
	"context"
)

type ctxKey struct{}

const MetadataKey = "x-request-id"

// FromContext extracts the request ID from the context.
func FromContext(ctx context.Context) string {
	if id, ok := ctx.Value(ctxKey{}).(string); ok {
		return id
	}
	return ""
}

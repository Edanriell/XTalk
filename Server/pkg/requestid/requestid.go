package requestid

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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

// WithRequestID returns a new context with the given request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// UnaryServerInterceptor injects or propagates a request ID in every gRPC call.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		requestID := extractFromMetadata(ctx)
		if requestID == "" {
			requestID = uuid.NewString()
		}
		ctx = WithRequestID(ctx, requestID)
		// Also set it in outgoing metadata for downstream calls
		ctx = metadata.AppendToOutgoingContext(ctx, MetadataKey, requestID)
		return handler(ctx, req)
	}
}

func extractFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get(MetadataKey)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

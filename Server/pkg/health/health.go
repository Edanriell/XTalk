package health

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Register adds the standard gRPC Health Checking service to the given server.
// Call SetServingStatus after the service is ready to accept traffic.
func Register(server *grpc.Server) *health.Server {
	hs := health.NewServer()
	healthpb.RegisterHealthServer(server, hs)
	return hs
}

// SetReady marks the given service (and the overall server) as SERVING.
func SetReady(hs *health.Server, serviceName string) {
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	hs.SetServingStatus(serviceName, healthpb.HealthCheckResponse_SERVING)
}

// SetNotReady marks the given service as NOT_SERVING (used during shutdown).
func SetNotReady(hs *health.Server, serviceName string) {
	hs.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	hs.SetServingStatus(serviceName, healthpb.HealthCheckResponse_NOT_SERVING)
}

// Check is a convenience function to check health from a client connection.
func Check(ctx context.Context, connection *grpc.ClientConn, service string) (healthpb.HealthCheckResponse_ServingStatus, error) {
	client := healthpb.NewHealthClient(connection)
	response, err := client.Check(ctx, &healthpb.HealthCheckRequest{Service: service})
	if err != nil {
		return healthpb.HealthCheckResponse_UNKNOWN, err
	}
	return response.Status, nil
}

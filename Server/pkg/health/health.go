package health

import (
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

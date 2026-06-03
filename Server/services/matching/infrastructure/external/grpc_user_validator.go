package external

import (
	"context"
	"fmt"
	"strings"

	"github.com/yourusername/connect/matching-service/application/interfaces"
	pb "github.com/yourusername/connect/proto/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// GRPCUserValidator implements UserValidator using gRPC calls to user-service
type GRPCUserValidator struct {
	conn   *grpc.ClientConn
	client pb.UserServiceClient
}

// NewGRPCUserValidator creates a new gRPC user validator
func NewGRPCUserValidator(userServiceAddr string) (interfaces.UserValidator, error) {
	if !strings.Contains(userServiceAddr, "://") {
		userServiceAddr = "passthrough:///" + userServiceAddr
	}
	conn, err := grpc.NewClient(userServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to user service: %w", err)
	}

	client := pb.NewUserServiceClient(conn)
	return &GRPCUserValidator{conn: conn, client: client}, nil
}

// UserExists checks if a user exists
func (v *GRPCUserValidator) UserExists(ctx context.Context, userID string) (bool, error) {
	req := &pb.GetUserRequest{UserId: userID}
	_, err := v.client.GetUser(ctx, req)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, fmt.Errorf("user-service lookup failed: %w", err)
	}
	return true, nil
}

// Close closes the underlying gRPC connection
func (v *GRPCUserValidator) Close() error {
	return v.conn.Close()
}

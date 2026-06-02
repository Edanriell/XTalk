package external

import (
	"context"
	"fmt"
	"strings"

	"github.com/yourusername/connect/matching-service/application/interfaces"
	pb "github.com/yourusername/connect/proto/chat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// GRPCChatCreator implements ChatCreator using gRPC calls to chat-service
type GRPCChatCreator struct {
	conn   *grpc.ClientConn
	client pb.ChatServiceClient
}

// NewGRPCChatCreator creates a new gRPC chat creator
func NewGRPCChatCreator(chatServiceAddr string) (interfaces.ChatCreator, error) {
	if !strings.Contains(chatServiceAddr, "://") {
		chatServiceAddr = "passthrough:///" + chatServiceAddr
	}
	conn, err := grpc.NewClient(chatServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to chat service: %w", err)
	}

	client := pb.NewChatServiceClient(conn)
	return &GRPCChatCreator{conn: conn, client: client}, nil
}

// CreateChat creates a new chat room between two users
func (c *GRPCChatCreator) CreateChat(ctx context.Context, user1ID, user2ID string, matchScore float64) (string, error) {
	req := &pb.CreateChatRequest{
		Participant1Id: user1ID,
		Participant2Id: user2ID,
		MatchScore:     float32(matchScore),
	}

	resp, err := c.client.CreateChat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create chat: %w", err)
	}

	if !resp.Success {
		return "", fmt.Errorf("chat creation failed: %s", resp.Message)
	}

	return resp.ChatId, nil
}

// Close closes the underlying gRPC connection
func (c *GRPCChatCreator) Close() error {
	return c.conn.Close()
}

package external

import (
	"context"
	"fmt"
	"strings"

	"github.com/yourusername/connect/message-service/application/interfaces"
	pb "github.com/yourusername/connect/proto/chat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// GRPCChatValidator implements ChatValidator using gRPC calls to chat-service
type GRPCChatValidator struct {
	conn   *grpc.ClientConn
	client pb.ChatServiceClient
}

// NewGRPCChatValidator creates a new gRPC chat validator
func NewGRPCChatValidator(chatServiceAddr string) (interfaces.ChatValidator, error) {
	if !strings.Contains(chatServiceAddr, "://") {
		chatServiceAddr = "passthrough:///" + chatServiceAddr
	}
	conn, err := grpc.NewClient(chatServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to chat service: %w", err)
	}

	client := pb.NewChatServiceClient(conn)
	return &GRPCChatValidator{conn: conn, client: client}, nil
}

// ChatExists checks if a chat room exists
func (v *GRPCChatValidator) ChatExists(ctx context.Context, chatID string) (bool, error) {
	req := &pb.GetChatRequest{ChatId: chatID, UserId: "system"}
	_, err := v.client.GetChat(ctx, req)
	if err != nil {
		if st, ok := status.FromError(err); ok && (st.Code() == codes.NotFound || st.Code() == codes.PermissionDenied) {
			// NotFound means chat doesn't exist; PermissionDenied means chat exists but user isn't participant
			if st.Code() == codes.NotFound {
				return false, nil
			}
			// PermissionDenied → chat exists but "system" isn't a participant
			return true, nil
		}
		return false, fmt.Errorf("chat service unavailable: %w", err)
	}
	return true, nil
}

// IsParticipant checks if a user is a participant in a chat
func (v *GRPCChatValidator) IsParticipant(ctx context.Context, chatID string, userID string) (bool, error) {
	req := &pb.GetChatRequest{ChatId: chatID, UserId: userID}
	resp, err := v.client.GetChat(ctx, req)
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.PermissionDenied {
			return false, nil
		}
		return false, err
	}

	if resp.Chat == nil {
		return false, nil
	}
	return resp.Chat.Participant1 == userID || resp.Chat.Participant2 == userID, nil
}

// Close closes the underlying gRPC connection
func (v *GRPCChatValidator) Close() error {
	return v.conn.Close()
}

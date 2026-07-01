package grpc

import (
	"context"

	"github.com/yourusername/connect/message-service/application/commands/delete_message"
	"github.com/yourusername/connect/message-service/application/commands/mark_as_read"
	"github.com/yourusername/connect/message-service/application/commands/send_message"
	"github.com/yourusername/connect/message-service/application/queries/get_messages"
	pb "github.com/yourusername/connect/proto/message"
	"google.golang.org/grpc/metadata"
)

// MessageGRPCService is the gRPC presentation layer
type MessageGRPCService struct {
	pb.UnimplementedMessageServiceServer
	sendMessageHandler   *send_message.Handler
	deleteMessageHandler *delete_message.Handler
	markAsReadHandler    *mark_as_read.Handler
	getMessagesHandler   *get_messages.Handler
}

// NewMessageGRPCService creates a new MessageGRPCService
func NewMessageGRPCService(
	sendMessageHandler *send_message.Handler,
	deleteMessageHandler *delete_message.Handler,
	markAsReadHandler *mark_as_read.Handler,
	getMessagesHandler *get_messages.Handler,
) *MessageGRPCService {
	return &MessageGRPCService{
		sendMessageHandler:   sendMessageHandler,
		deleteMessageHandler: deleteMessageHandler,
		markAsReadHandler:    markAsReadHandler,
		getMessagesHandler:   getMessagesHandler,
	}
}

// SendMessage sends a new message
func (s *MessageGRPCService) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	cmd := send_message.Command{
		ChatID:      req.ChatId,
		SenderID:    req.SenderId,
		MessageType: req.MessageType,
		Content:     req.Content,
		Metadata:    req.Metadata,
	}

	result, err := s.sendMessageHandler.Handle(ctx, cmd)
	if err != nil {
		return &pb.SendMessageResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.SendMessageResponse{
		MessageId: result.MessageID,
		Success:   true,
		Message:   "Message sent successfully",
		Timestamp: result.CreatedAt.Unix(),
	}, nil
}

// GetMessages retrieves messages for a chat
func (s *MessageGRPCService) GetMessages(ctx context.Context, req *pb.GetMessagesRequest) (*pb.GetMessagesResponse, error) {
	// Extract caller user ID from gRPC metadata for authorization.
	var userID string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-user-id"); len(vals) > 0 {
			userID = vals[0]
		}
	}

	query := get_messages.Query{
		ChatID: req.ChatId,
		UserID: userID,
		Limit:  int(req.Limit),
		Offset: int(req.Offset),
	}

	result, err := s.getMessagesHandler.Handle(ctx, query)
	if err != nil {
		return &pb.GetMessagesResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	messages := make([]*pb.Message, 0, len(result.Messages))
	for _, dto := range result.Messages {
		msg := &pb.Message{
			Id:          dto.ID,
			ChatId:      dto.ChatID,
			SenderId:    dto.SenderID,
			MessageType: dto.MessageType,
			Content:     dto.Content,
			Metadata:    dto.Metadata,
			IsRead:      dto.IsRead,
			CreatedAt:   dto.CreatedAt.Unix(),
		}

		if dto.ReadAt != nil {
			msg.ReadAt = dto.ReadAt.Unix()
		}
		if dto.DeletedAt != nil {
			msg.DeletedAt = dto.DeletedAt.Unix()
		}

		messages = append(messages, msg)
	}

	return &pb.GetMessagesResponse{
		Success:  true,
		Message:  "Messages retrieved successfully",
		Messages: messages,
	}, nil
}

// DeleteMessage deletes a message
func (s *MessageGRPCService) DeleteMessage(ctx context.Context, req *pb.DeleteMessageRequest) (*pb.DeleteMessageResponse, error) {
	cmd := delete_message.Command{
		MessageID: req.MessageId,
		UserID:    req.UserId,
	}

	result, err := s.deleteMessageHandler.Handle(ctx, cmd)
	if err != nil {
		return &pb.DeleteMessageResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.DeleteMessageResponse{
		MessageId: result.MessageID,
		Success:   true,
		Message:   "Message deleted successfully",
	}, nil
}

// MarkAsRead marks a message as read
func (s *MessageGRPCService) MarkAsRead(ctx context.Context, req *pb.MarkAsReadRequest) (*pb.MarkAsReadResponse, error) {
	cmd := mark_as_read.Command{
		MessageID: req.MessageId,
		UserID:    req.UserId,
	}

	result, err := s.markAsReadHandler.Handle(ctx, cmd)
	if err != nil {
		return &pb.MarkAsReadResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.MarkAsReadResponse{
		MessageId: result.MessageID,
		Success:   true,
		Message:   "Message marked as read",
		ReadAt:    result.ReadAt.Unix(),
	}, nil
}

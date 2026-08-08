package grpc

import (
	"context"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "XTalk/proto/chat"
	"XTalk/services/chat/application/create_chat"
	"XTalk/services/chat/application/end_chat"
	"XTalk/services/chat/application/get_chat"
	"XTalk/services/chat/application/get_user_chats"
)

// ChatGRPCService is the gRPC presentation layer
type ChatGRPCService struct {
	pb.UnimplementedChatServiceServer
	createChatHandler   *create_chat.Handler
	endChatHandler      *end_chat.Handler
	getChatHandler      *get_chat.Handler
	getUserChatsHandler *get_user_chats.Handler
}

// NewChatGRPCService creates a new ChatGRPCService
func NewChatGRPCService(
	createChatHandler *create_chat.Handler,
	endChatHandler *end_chat.Handler,
	getChatHandler *get_chat.Handler,
	getUserChatsHandler *get_user_chats.Handler,
) *ChatGRPCService {
	return &ChatGRPCService{
		createChatHandler:   createChatHandler,
		endChatHandler:      endChatHandler,
		getChatHandler:      getChatHandler,
		getUserChatsHandler: getUserChatsHandler,
	}
}

// CreateChat creates a new chat
func (s *ChatGRPCService) CreateChat(ctx context.Context, req *pb.CreateChatRequest) (*pb.CreateChatResponse, error) {
	cmd := create_chat.Command{
		Participant1: req.Participant1Id,
		Participant2: req.Participant2Id,
		MatchScore:   float64(req.MatchScore),
	}

	result, err := s.createChatHandler.Handle(ctx, cmd)
	if err != nil {
		return &pb.CreateChatResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.CreateChatResponse{
		ChatId:       result.ChatID,
		Participant1: result.Participant1,
		Participant2: result.Participant2,
		MatchScore:   float32(result.MatchScore),
		Success:      result.Success,
		Message:      result.Message,
	}, nil
}

// GetChat retrieves a chat by ID
func (s *ChatGRPCService) GetChat(ctx context.Context, req *pb.GetChatRequest) (*pb.GetChatResponse, error) {
	query := get_chat.Query{
		ChatID: req.ChatId,
		UserID: req.UserId,
	}

	dto, err := s.getChatHandler.Handle(ctx, query)
	if err != nil {
		return &pb.GetChatResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	response := &pb.GetChatResponse{
		Success: true,
		Message: "Chat retrieved successfully",
		Chat: &pb.Chat{
			Id:           dto.ID,
			Participant1: dto.Participant1,
			Participant2: dto.Participant2,
			Status:       dto.Status,
			MatchScore:   float32(dto.MatchScore),
			CreatedAt:    timestamppb.New(dto.CreatedAt),
			UpdatedAt:    timestamppb.New(dto.UpdatedAt),
		},
	}

	if dto.EndedAt != nil {
		response.Chat.EndedAt = timestamppb.New(*dto.EndedAt)
	}

	return response, nil
}

// GetUserChats retrieves all chats for a user
func (s *ChatGRPCService) GetUserChats(ctx context.Context, req *pb.GetUserChatsRequest) (*pb.GetUserChatsResponse, error) {
	query := get_user_chats.Query{
		UserID: req.UserId,
		Limit:  int(req.Limit),
		Offset: int(req.Offset),
	}

	dtoList, err := s.getUserChatsHandler.Handle(ctx, query)
	if err != nil {
		return &pb.GetUserChatsResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	chats := make([]*pb.Chat, 0, len(dtoList.Chats))
	for _, dto := range dtoList.Chats {
		chat := &pb.Chat{
			Id:           dto.ID,
			Participant1: dto.Participant1,
			Participant2: dto.Participant2,
			Status:       dto.Status,
			MatchScore:   float32(dto.MatchScore),
			CreatedAt:    timestamppb.New(dto.CreatedAt),
			UpdatedAt:    timestamppb.New(dto.UpdatedAt),
		}

		if dto.EndedAt != nil {
			chat.EndedAt = timestamppb.New(*dto.EndedAt)
		}

		chats = append(chats, chat)
	}

	return &pb.GetUserChatsResponse{
		Success: true,
		Message: "Chats retrieved successfully",
		Chats:   chats,
		Total:   int32(dtoList.Total),
	}, nil
}

// EndChat ends a chat
func (s *ChatGRPCService) EndChat(ctx context.Context, req *pb.EndChatRequest) (*pb.EndChatResponse, error) {
	cmd := end_chat.Command{
		ChatID: req.ChatId,
		UserID: req.UserId,
	}

	result, err := s.endChatHandler.Handle(ctx, cmd)
	if err != nil {
		return &pb.EndChatResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.EndChatResponse{
		ChatId:  result.ChatID,
		Success: result.Success,
		Message: result.Message,
	}, nil
}

// ---------------------  Room-based RPCs  ---------------------
// These map the Room abstraction (used by the API gateway) onto the
// underlying 1:1 Chat domain model.

// CreateRoom creates a chat between the creator and the first participant.
func (s *ChatGRPCService) CreateRoom(ctx context.Context, req *pb.CreateRoomRequest) (*pb.CreateRoomResponse, error) {
	if len(req.ParticipantIds) == 0 {
		return &pb.CreateRoomResponse{Success: false, Message: "at least one participant required"}, nil
	}

	// In the 1:1 model, participant1 = creator, participant2 = first participant.
	cmd := create_chat.Command{
		Participant1: req.CreatorId,
		Participant2: req.ParticipantIds[0],
	}

	result, err := s.createChatHandler.Handle(ctx, cmd)
	if err != nil {
		return &pb.CreateRoomResponse{Success: false, Message: err.Error()}, nil
	}

	return &pb.CreateRoomResponse{
		Success: result.Success,
		Message: result.Message,
		Room: &pb.ChatRoom{
			Id:             result.ChatID,
			Name:           req.Name,
			Type:           req.Type,
			CreatorId:      req.CreatorId,
			ParticipantIds: []string{result.Participant1, result.Participant2},
		},
	}, nil
}

// GetRoom retrieves a chat and returns it as a ChatRoom.
// The calling user's ID is extracted from gRPC metadata ("x-user-id").
func (s *ChatGRPCService) GetRoom(ctx context.Context, req *pb.GetRoomRequest) (*pb.GetRoomResponse, error) {
	var userID string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-user-id"); len(vals) > 0 {
			userID = vals[0]
		}
	}

	query := get_chat.Query{
		ChatID: req.RoomId,
		UserID: userID,
	}

	dto, err := s.getChatHandler.Handle(ctx, query)
	if err != nil {
		return &pb.GetRoomResponse{Success: false, Message: err.Error()}, nil
	}

	return &pb.GetRoomResponse{
		Success: true,
		Message: "Room retrieved successfully",
		Room: &pb.ChatRoom{
			Id:             dto.ID,
			ParticipantIds: []string{dto.Participant1, dto.Participant2},
		},
	}, nil
}

// GetUserRooms lists all chats for a user, presented as ChatRoom objects.
func (s *ChatGRPCService) GetUserRooms(ctx context.Context, req *pb.GetUserRoomsRequest) (*pb.GetUserRoomsResponse, error) {
	query := get_user_chats.Query{
		UserID: req.UserId,
		Limit:  50,
	}

	dtoList, err := s.getUserChatsHandler.Handle(ctx, query)
	if err != nil {
		return &pb.GetUserRoomsResponse{Success: false, Message: err.Error()}, nil
	}

	rooms := make([]*pb.ChatRoom, 0, len(dtoList.Chats))
	for _, dto := range dtoList.Chats {
		rooms = append(rooms, &pb.ChatRoom{
			Id:             dto.ID,
			ParticipantIds: []string{dto.Participant1, dto.Participant2},
		})
	}

	return &pb.GetUserRoomsResponse{
		Success: true,
		Message: "Rooms retrieved successfully",
		Rooms:   rooms,
	}, nil
}

// DeleteRoom ends the underlying chat.
func (s *ChatGRPCService) DeleteRoom(ctx context.Context, req *pb.DeleteRoomRequest) (*pb.DeleteRoomResponse, error) {
	cmd := end_chat.Command{
		ChatID: req.RoomId,
		UserID: req.UserId,
	}

	result, err := s.endChatHandler.Handle(ctx, cmd)
	if err != nil {
		return &pb.DeleteRoomResponse{Success: false, Message: err.Error()}, nil
	}

	return &pb.DeleteRoomResponse{
		Success: result.Success,
		Message: result.Message,
	}, nil
}

// JoinRoom is a no-op in the 1:1 chat model; room membership is implicit.
func (s *ChatGRPCService) JoinRoom(_ context.Context, _ *pb.JoinRoomRequest) (*pb.JoinRoomResponse, error) {
	return &pb.JoinRoomResponse{Success: true, Message: "joined"}, nil
}

// LeaveRoom is a no-op in the 1:1 chat model.
func (s *ChatGRPCService) LeaveRoom(_ context.Context, _ *pb.LeaveRoomRequest) (*pb.LeaveRoomResponse, error) {
	return &pb.LeaveRoomResponse{Success: true, Message: "left"}, nil
}

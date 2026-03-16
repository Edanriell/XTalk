package grpc

import (
	"XTalk/services/user/application/users/create_user"
	"XTalk/services/user/application/users/delete_user"
	"XTalk/services/user/application/users/get_user"
	"XTalk/services/user/application/users/get_user_by_email"
	"XTalk/services/user/application/users/update_status"
	"XTalk/services/user/application/users/update_user"
	"context"
)

// UserGRPCService is the gRPC transport adapter.
type UserGRPCService struct {
	pb.UnimplementedUserServiceServer
	createUser     *create_user.Handler
	updateUser     *update_user.Handler
	updateStatus   *update_status.Handler
	deleteUser     *delete_user.Handler
	getUser        *get_user.Handler
	getUserByEmail *get_user_by_email.Handler
}

func NewUserGRPCService(
	createUser *create_user.Handler,
	updateUser *update_user.Handler,
	updateStatus *update_status.Handler,
	deleteUser *delete_user.Handler,
	getUser *get_user.Handler,
	getUserByEmail *get_user_by_email.Handler,
) *UserGRPCService {
	return &UserGRPCService{
		createUser:     createUser,
		updateUser:     updateUser,
		updateStatus:   updateStatus,
		deleteUser:     deleteUser,
		getUser:        getUser,
		getUserByEmail: getUserByEmail,
	}
}

func (s *UserGRPCService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	err := s.createUser.Handle(ctx, create_user.Command{
		UserID:   req.UserId,
		Username: req.Username,
		Email:    req.Email,
	})
	if err != nil {
		return &pb.CreateUserResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.CreateUserResponse{Success: true, Message: "User created successfully"}, nil
}

func (s *UserGRPCService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	dto, err := s.getUser.Handle(ctx, get_user.Query{UserID: req.UserId})
	if err != nil {
		return &pb.GetUserResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.GetUserResponse{
		Success: true,
		Message: "User retrieved successfully",
		User:    dtoToProto(dto),
	}, nil
}

func (s *UserGRPCService) GetUserByEmail(ctx context.Context, req *pb.GetUserByEmailRequest) (*pb.GetUserByEmailResponse, error) {
	dto, err := s.getUserByEmail.Handle(ctx, get_user_by_email.Query{Email: req.Email})
	if err != nil {
		return &pb.GetUserByEmailResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.GetUserByEmailResponse{
		Success: true,
		Message: "User retrieved successfully",
		User:    emailDTOToProto(dto),
	}, nil
}

func (s *UserGRPCService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	result, err := s.updateUser.Handle(ctx, update_user.Command{
		UserID:    req.UserId,
		Username:  req.Username,
		Bio:       req.Bio,
		Age:       int(req.Age),
		Gender:    req.Gender,
		Country:   req.Country,
		Language:  req.Language,
		Interests: req.Interests,
		AvatarURL: req.AvatarUrl,
	})
	if err != nil {
		return &pb.UpdateUserResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.UpdateUserResponse{
		UserId:  result.UserID,
		Success: result.Success,
		Message: result.Message,
	}, nil
}

func (s *UserGRPCService) UpdateStatus(ctx context.Context, req *pb.UpdateStatusRequest) (*pb.UpdateStatusResponse, error) {
	result, err := s.updateStatus.Handle(ctx, update_status.Command{
		UserID: req.UserId,
		Status: req.Status,
	})
	if err != nil {
		return &pb.UpdateStatusResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.UpdateStatusResponse{
		UserId:  result.UserID,
		Status:  result.Status,
		Success: result.Success,
		Message: result.Message,
	}, nil
}

func (s *UserGRPCService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	result, err := s.deleteUser.Handle(ctx, delete_user.Command{UserID: req.UserId})
	if err != nil {
		return &pb.DeleteUserResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.DeleteUserResponse{Success: result.Success, Message: result.Message}, nil
}

// --- mapping helpers ---

func dtoToProto(d *get_user.DTO) *pb.User {
	return &pb.User{
		Id:        d.ID,
		Username:  d.Username,
		Email:     d.Email,
		Age:       int32(d.Age),
		Gender:    d.Gender,
		Country:   d.Country,
		Language:  d.Language,
		Interests: d.Interests,
		Status:    d.Status,
		Bio:       d.Bio,
		AvatarUrl: d.AvatarURL,
		CreatedAt: d.CreatedAt.Unix(),
		UpdatedAt: d.UpdatedAt.Unix(),
		LastSeen:  d.LastSeen.Unix(),
		IsActive:  d.IsActive,
	}
}

func emailDTOToProto(d *get_user_by_email.DTO) *pb.User {
	return &pb.User{
		Id:        d.ID,
		Username:  d.Username,
		Email:     d.Email,
		Age:       int32(d.Age),
		Gender:    d.Gender,
		Country:   d.Country,
		Language:  d.Language,
		Interests: d.Interests,
		Status:    d.Status,
		Bio:       d.Bio,
		AvatarUrl: d.AvatarURL,
		CreatedAt: d.CreatedAt.Unix(),
		UpdatedAt: d.UpdatedAt.Unix(),
		LastSeen:  d.LastSeen.Unix(),
		IsActive:  d.IsActive,
	}
}

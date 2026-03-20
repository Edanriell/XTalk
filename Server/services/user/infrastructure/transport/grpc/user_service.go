package grpc

import (
	"context"
	"errors"

	userpb "XTalk/proto/user"
	"XTalk/services/user/application/users/create_user"
	"XTalk/services/user/application/users/delete_user"
	"XTalk/services/user/application/users/get_user"
	"XTalk/services/user/application/users/get_user_by_email"
	"XTalk/services/user/application/users/readmodel"
	"XTalk/services/user/application/users/update_status"
	"XTalk/services/user/application/users/update_user"
	domainusers "XTalk/services/user/domain/users"

	"go.uber.org/zap"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

type application interface {
	Create(context.Context, create_user.Command) error
	Delete(context.Context, delete_user.Command) error
	Get(context.Context, get_user.Query) (*get_user.Response, error)
	GetByEmail(context.Context, get_user_by_email.Query) (*get_user_by_email.Response, error)
	UpdateStatus(context.Context, update_status.Command) (*update_status.Result, error)
	Update(context.Context, update_user.Command) (*update_user.Result, error)
}

// UserService is the thin gRPC adapter for user application use cases.
type UserService struct {
	userpb.UnimplementedUserServiceServer
	application application
	log         *zap.Logger
}

func NewUserService(application application, log *zap.Logger) *UserService {
	return &UserService{application: application, log: log}
}

func (s *UserService) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	err := s.application.Create(ctx, create_user.Command{
		UserID: req.GetUserId(), Username: req.GetUsername(), Email: req.GetEmail(),
	})
	if err != nil {
		return nil, s.toGRPCError(err)
	}
	return &userpb.CreateUserResponse{Success: true, Message: "User created successfully"}, nil
}

func (s *UserService) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.GetUserResponse, error) {
	user, err := s.application.Get(ctx, get_user.Query{UserID: req.GetUserId()})
	if err != nil {
		return nil, s.toGRPCError(err)
	}
	return &userpb.GetUserResponse{
		Success: true, Message: "User retrieved successfully", User: userToProto(user),
	}, nil
}

func (s *UserService) GetUserByEmail(ctx context.Context, req *userpb.GetUserByEmailRequest) (*userpb.GetUserByEmailResponse, error) {
	user, err := s.application.GetByEmail(ctx, get_user_by_email.Query{Email: req.GetEmail()})
	if err != nil {
		return nil, s.toGRPCError(err)
	}
	return &userpb.GetUserByEmailResponse{
		Success: true, Message: "User retrieved successfully", User: userToProto(user),
	}, nil
}

func (s *UserService) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	result, err := s.application.Update(ctx, update_user.Command{
		UserID: req.GetUserId(), Username: req.GetUsername(), Bio: req.GetBio(),
		Age: int(req.GetAge()), Gender: req.GetGender(), Country: req.GetCountry(),
		Language: req.GetLanguage(), Interests: req.GetInterests(), AvatarURL: req.GetAvatarUrl(),
	})
	if err != nil {
		return nil, s.toGRPCError(err)
	}
	return &userpb.UpdateUserResponse{
		UserId: result.UserID, Success: true, Message: "User updated successfully",
	}, nil
}

func (s *UserService) UpdateStatus(ctx context.Context, req *userpb.UpdateStatusRequest) (*userpb.UpdateStatusResponse, error) {
	result, err := s.application.UpdateStatus(ctx, update_status.Command{
		UserID: req.GetUserId(), Status: req.GetStatus(),
	})
	if err != nil {
		return nil, s.toGRPCError(err)
	}
	return &userpb.UpdateStatusResponse{
		UserId: result.UserID, Status: result.Status, Success: true, Message: "Status updated successfully",
	}, nil
}

func (s *UserService) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	err := s.application.Delete(ctx, delete_user.Command{UserID: req.GetUserId()})
	if err != nil {
		return nil, s.toGRPCError(err)
	}
	return &userpb.DeleteUserResponse{Success: true, Message: "User deleted successfully"}, nil
}

func userToProto(user *readmodel.User) *userpb.User {
	return &userpb.User{
		Id: user.ID, Username: user.Username, Email: user.Email, Age: int32(user.Age),
		Gender: user.Gender, Country: user.Country, Language: user.Language,
		Interests: user.Interests, Status: user.Status, Bio: user.Bio,
		AvatarUrl: user.AvatarURL, CreatedAt: user.CreatedAt.Unix(),
		UpdatedAt: user.UpdatedAt.Unix(), LastSeen: user.LastSeen.Unix(), IsActive: user.IsActive,
	}
}

func (s *UserService) toGRPCError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return grpcstatus.Error(grpccodes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return grpcstatus.Error(grpccodes.DeadlineExceeded, "request deadline exceeded")
	case errors.Is(err, domainusers.ErrUserNotFound):
		return grpcstatus.Error(grpccodes.NotFound, err.Error())
	case errors.Is(err, domainusers.ErrUserAlreadyExists):
		return grpcstatus.Error(grpccodes.AlreadyExists, err.Error())
	case errors.Is(err, domainusers.ErrUserInactive):
		return grpcstatus.Error(grpccodes.FailedPrecondition, err.Error())
	case domainusers.IsValidationError(err):
		return grpcstatus.Error(grpccodes.InvalidArgument, err.Error())
	default:
		s.log.Error("user RPC failed", zap.Error(err))
		return grpcstatus.Error(grpccodes.Internal, "internal server error")
	}
}

var _ userpb.UserServiceServer = (*UserService)(nil)

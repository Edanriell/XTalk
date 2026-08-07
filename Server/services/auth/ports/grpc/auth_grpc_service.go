package grpc

import (
	pb "XTalk/proto/auth"
	"XTalk/services/auth/application/login"
	"XTalk/services/auth/application/logout"
	"XTalk/services/auth/application/refresh_token"
	"XTalk/services/auth/application/register"
	"XTalk/services/auth/application/validate_token"
	"context"
)

// AuthGRPCService is the gRPC presentation layer
type AuthGRPCService struct {
	pb.UnimplementedAuthServiceServer
	registerHandler      *register.Handler
	loginHandler         *login.Handler
	validateTokenHandler *validate_token.Handler
	refreshTokenHandler  *refresh_token.Handler
	logoutHandler        *logout.Handler
}

func NewAuthGRPCService(
	registerHandler *register.Handler,
	loginHandler *login.Handler,
	validateTokenHandler *validate_token.Handler,
	refreshTokenHandler *refresh_token.Handler,
	logoutHandler *logout.Handler,
) *AuthGRPCService {
	return &AuthGRPCService{
		registerHandler:      registerHandler,
		loginHandler:         loginHandler,
		validateTokenHandler: validateTokenHandler,
		refreshTokenHandler:  refreshTokenHandler,
		logoutHandler:        logoutHandler,
	}
}

func (s *AuthGRPCService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	cmd := register.Command{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}

	result, err := s.registerHandler.Handle(ctx, cmd)
	if err != nil {
		return &pb.RegisterResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.RegisterResponse{
		UserId:       result.UserID,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		Success:      true,
		Message:      "User registered successfully",
	}, nil
}

func (s *AuthGRPCService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	cmd := login.Command{
		Email:    req.Email,
		Password: req.Password,
	}

	result, err := s.loginHandler.Handle(ctx, cmd)
	if err != nil {
		return &pb.LoginResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.LoginResponse{
		UserId:       result.UserID,
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		Success:      true,
		Message:      "Login successful",
	}, nil
}

func (s *AuthGRPCService) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	query := validate_token.Query{
		Token: req.Token,
	}

	result, err := s.validateTokenHandler.Handle(ctx, query)
	if err != nil {
		return &pb.ValidateTokenResponse{
			Valid:   false,
			Message: err.Error(),
		}, nil
	}

	if !result.Valid {
		return &pb.ValidateTokenResponse{
			Valid:   false,
			Message: "Token is invalid",
		}, nil
	}

	return &pb.ValidateTokenResponse{
		UserId:  result.UserID,
		Email:   result.Email,
		Valid:   true,
		Message: "Token is valid",
	}, nil
}

func (s *AuthGRPCService) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	cmd := refresh_token.Command{
		RefreshToken: req.RefreshToken,
	}

	result, err := s.refreshTokenHandler.Handle(ctx, cmd)
	if err != nil {
		return &pb.RefreshTokenResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.RefreshTokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		Success:      true,
		Message:      "Token refreshed successfully",
	}, nil
}

func (s *AuthGRPCService) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	cmd := logout.Command{
		UserID: req.UserId,
		Token:  req.Token,
	}

	result, err := s.logoutHandler.Handle(ctx, cmd)
	if err != nil {
		return &pb.LogoutResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	if !result.Success {
		return &pb.LogoutResponse{
			Success: false,
			Message: "Logout failed",
		}, nil
	}

	return &pb.LogoutResponse{
		Success: true,
		Message: "Logged out successfully",
	}, nil
}

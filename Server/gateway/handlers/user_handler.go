package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/yourusername/connect/api-gateway/circuitbreaker"
	"github.com/yourusername/connect/api-gateway/config"
	userpb "github.com/yourusername/connect/proto/user"
)

type UserHandler struct {
	conn        *grpc.ClientConn
	userClient  userpb.UserServiceClient
	log         *zap.Logger
	grpcTimeout time.Duration
}

func NewUserHandler(cfg *config.Config, log *zap.Logger, cbr *circuitbreaker.Registry) *UserHandler {
	opts := append([]grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}, cbr.DialOptions("UserService")...)

	userConn, err := grpc.NewClient(cfg.UserServiceAddr, opts...)
	if err != nil {
		log.Fatal("failed to connect to user service", zap.Error(err))
	}

	return &UserHandler{
		conn:        userConn,
		userClient:  userpb.NewUserServiceClient(userConn),
		log:         log.Named("user"),
		grpcTimeout: cfg.GRPCTimeout,
	}
}

// Close releases the underlying gRPC connection.
func (h *UserHandler) Close() error { return h.conn.Close() }

// ensureUserProfile creates a user profile in the user-service if it doesn't exist.
// This handles the case where the RabbitMQ event from auth-service was lost or delayed.
func (h *UserHandler) ensureUserProfile(ctx context.Context, userID, email string) bool {
	resp, err := h.userClient.CreateUser(ctx, &userpb.CreateUserRequest{
		UserId:   userID,
		Username: userID, // placeholder; user can update later
		Email:    email,
	})
	if err != nil {
		h.log.Warn("auto-provision user profile failed (transport)", zap.String("user_id", userID), zap.Error(err))
		return false
	}
	if !resp.Success {
		h.log.Warn("auto-provision user profile failed", zap.String("user_id", userID), zap.String("message", resp.Message))
		return false
	}
	h.log.Info("auto-provisioned user profile", zap.String("user_id", userID))
	return true
}

func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID := c.GetString("userID")
	email := c.GetString("email")
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := h.userClient.GetUser(ctx, &userpb.GetUserRequest{UserId: userID})
	if err != nil {
		h.log.Error("get user call failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	if !resp.Success && resp.Message == "user not found" && email != "" {
		h.ensureUserProfile(ctx, userID, email)
		resp, err = h.userClient.GetUser(ctx, &userpb.GetUserRequest{UserId: userID})
		if err != nil {
			h.log.Error("get user call failed after auto-provision", zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
			return
		}
	}

	if !resp.Success {
		c.JSON(http.StatusNotFound, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID := c.GetString("userID")
	email := c.GetString("email")
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Username  string   `json:"username"`
		AvatarURL string   `json:"avatar_url"`
		Bio       string   `json:"bio"`
		Age       int32    `json:"age"`
		Gender    string   `json:"gender"`
		Country   string   `json:"country"`
		Language  string   `json:"language"`
		Interests []string `json:"interests"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := h.userClient.UpdateUser(ctx, &userpb.UpdateUserRequest{
		UserId:    userID,
		Username:  req.Username,
		AvatarUrl: req.AvatarURL,
		Bio:       req.Bio,
		Age:       req.Age,
		Gender:    req.Gender,
		Country:   req.Country,
		Language:  req.Language,
		Interests: req.Interests,
	})
	if err != nil {
		h.log.Error("update user call failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	if !resp.Success && resp.Message == "user not found" && email != "" {
		h.ensureUserProfile(ctx, userID, email)
		resp, err = h.userClient.UpdateUser(ctx, &userpb.UpdateUserRequest{
			UserId:    userID,
			Username:  req.Username,
			AvatarUrl: req.AvatarURL,
			Bio:       req.Bio,
			Age:       req.Age,
			Gender:    req.Gender,
			Country:   req.Country,
			Language:  req.Language,
			Interests: req.Interests,
		})
		if err != nil {
			h.log.Error("update user call failed after auto-provision", zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
			return
		}
	}

	c.JSON(http.StatusOK, resp)
}

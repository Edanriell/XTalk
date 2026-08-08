package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	"XTalk/gateway/application"
	userpb "XTalk/proto/user"
)

type UserHandler struct {
	userClient  userpb.UserServiceClient
	log         *zap.Logger
	grpcTimeout time.Duration
}

func NewUserHandler(client userpb.UserServiceClient, cfg *application.Config, log *zap.Logger) *UserHandler {
	return &UserHandler{
		userClient:  client,
		log:         log.Named("user"),
		grpcTimeout: cfg.GRPCTimeout,
	}
}

func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID := c.GetString("userID")
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

	if !resp.Success {
		c.JSON(http.StatusNotFound, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID := c.GetString("userID")
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

	c.JSON(http.StatusOK, resp)
}

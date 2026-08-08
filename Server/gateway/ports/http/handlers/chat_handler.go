package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	"XTalk/gateway/application"
	chatpb "XTalk/proto/chat"
)

type ChatHandler struct {
	chatClient  chatpb.ChatServiceClient
	log         *zap.Logger
	grpcTimeout time.Duration
}

func NewChatHandler(client chatpb.ChatServiceClient, cfg *application.Config, log *zap.Logger) *ChatHandler {
	return &ChatHandler{
		chatClient:  client,
		log:         log.Named("chat"),
		grpcTimeout: cfg.GRPCTimeout,
	}
}

func (h *ChatHandler) GetUserRooms(c *gin.Context) {
	userID := c.GetString("userID")
	token := extractToken(c)

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := h.chatClient.GetUserRooms(ctx, &chatpb.GetUserRoomsRequest{UserId: userID})
	if err != nil {
		h.log.Error("get user rooms failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ChatHandler) CreateRoom(c *gin.Context) {
	userID := c.GetString("userID")
	token := extractToken(c)

	var req struct {
		Name           string   `json:"name"`
		Type           string   `json:"type"`
		ParticipantIDs []string `json:"participant_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate inputs
	if len(req.Name) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room name too long"})
		return
	}
	validTypes := map[string]bool{"direct": true, "group": true}
	if req.Type != "" && !validTypes[req.Type] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be 'direct' or 'group'"})
		return
	}
	if req.Type == "group" && strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group rooms require a name"})
		return
	}
	if len(req.ParticipantIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one participant required"})
		return
	}
	if len(req.ParticipantIDs) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "too many participants"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := h.chatClient.CreateRoom(ctx, &chatpb.CreateRoomRequest{
		Name:           req.Name,
		Type:           req.Type,
		CreatorId:      userID,
		ParticipantIds: req.ParticipantIDs,
	})
	if err != nil {
		h.log.Error("create room failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	if resp.Success {
		c.JSON(http.StatusCreated, resp)
	} else {
		c.JSON(http.StatusBadRequest, resp)
	}
}

func (h *ChatHandler) GetRoom(c *gin.Context) {
	roomID := c.Param("id")
	userID := c.GetString("userID")
	token := extractToken(c)

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	md := metadata.New(map[string]string{
		"authorization": "Bearer " + token,
		"x-user-id":     userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := h.chatClient.GetRoom(ctx, &chatpb.GetRoomRequest{RoomId: roomID})
	if err != nil {
		h.log.Error("get room failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusNotFound, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ChatHandler) DeleteRoom(c *gin.Context) {
	roomID := c.Param("id")
	userID := c.GetString("userID")
	token := extractToken(c)

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := h.chatClient.DeleteRoom(ctx, &chatpb.DeleteRoomRequest{
		RoomId: roomID,
		UserId: userID,
	})
	if err != nil {
		h.log.Error("delete room failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusForbidden, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}

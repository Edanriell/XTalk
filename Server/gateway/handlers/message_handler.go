package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/yourusername/connect/api-gateway/circuitbreaker"
	"github.com/yourusername/connect/api-gateway/config"
	msgpb "github.com/yourusername/connect/proto/message"
)

type MessageHandler struct {
	conn        *grpc.ClientConn
	msgClient   msgpb.MessageServiceClient
	log         *zap.Logger
	grpcTimeout time.Duration
}

func NewMessageHandler(cfg *config.Config, log *zap.Logger, cbr *circuitbreaker.Registry) *MessageHandler {
	opts := append([]grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}, cbr.DialOptions("MessageService")...)

	msgConn, err := grpc.NewClient(cfg.MessageServiceAddr, opts...)
	if err != nil {
		log.Fatal("failed to connect to message service", zap.Error(err))
	}

	return &MessageHandler{
		conn:        msgConn,
		msgClient:   msgpb.NewMessageServiceClient(msgConn),
		log:         log.Named("message"),
		grpcTimeout: cfg.GRPCTimeout,
	}
}

// Close releases the underlying gRPC connection.
func (h *MessageHandler) Close() error { return h.conn.Close() }

func (h *MessageHandler) GetMessages(c *gin.Context) {
	token := extractToken(c)
	userID := c.GetString("userID")

	chatID := c.Query("chat_id")
	if chatID == "" {
		chatID = c.Query("room_id")
	}
	if chatID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chat_id is required"})
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit < 1 {
		limit = 50
	} else if limit > 200 {
		limit = 200
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}
	const maxOffset = 1_000_000 // guard against int32 overflow
	if offset > maxOffset {
		offset = maxOffset
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	md := metadata.New(map[string]string{
		"authorization": "Bearer " + token,
		"x-user-id":     userID,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := h.msgClient.GetMessages(ctx, &msgpb.GetMessagesRequest{
		ChatId: chatID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		h.log.Error("get messages failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *MessageHandler) SendMessage(c *gin.Context) {
	userID := c.GetString("userID")
	token := extractToken(c)

	var req struct {
		RoomID  string `json:"room_id" binding:"required"`
		Content string `json:"content" binding:"required"`
		Type    string `json:"type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	const maxContentLen = 10_000 // matches message-service MaxContentLength
	if len(req.Content) > maxContentLen {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message content too long"})
		return
	}

	if req.Type == "" {
		req.Type = "text"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := h.msgClient.SendMessage(ctx, &msgpb.SendMessageRequest{
		ChatId:      req.RoomID,
		SenderId:    userID,
		Content:     req.Content,
		MessageType: req.Type,
	})
	if err != nil {
		h.log.Error("send message failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	if resp.Success {
		c.JSON(http.StatusCreated, resp)
	} else {
		c.JSON(http.StatusBadRequest, resp)
	}
}

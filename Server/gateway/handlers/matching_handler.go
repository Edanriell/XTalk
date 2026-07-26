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
	matchingpb "github.com/yourusername/connect/proto/matching"
)

type MatchingHandler struct {
	conn           *grpc.ClientConn
	matchingClient matchingpb.MatchingServiceClient
	log            *zap.Logger
	grpcTimeout    time.Duration
}

func NewMatchingHandler(cfg *config.Config, log *zap.Logger, cbr *circuitbreaker.Registry) *MatchingHandler {
	opts := append([]grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}, cbr.DialOptions("MatchingService")...)

	matchingConn, err := grpc.NewClient(cfg.MatchingServiceAddr, opts...)
	if err != nil {
		log.Fatal("failed to connect to matching service", zap.Error(err))
	}

	return &MatchingHandler{
		conn:           matchingConn,
		matchingClient: matchingpb.NewMatchingServiceClient(matchingConn),
		log:            log.Named("matching"),
		grpcTimeout:    cfg.GRPCTimeout,
	}
}

// Close releases the underlying gRPC connection.
func (h *MatchingHandler) Close() error { return h.conn.Close() }

func (h *MatchingHandler) StartMatching(c *gin.Context) {
	userID := c.GetString("userID")
	token := extractToken(c)

	var req struct {
		Age       int32    `json:"age"`
		MinAge    *int32   `json:"min_age"`
		MaxAge    *int32   `json:"max_age"`
		Interests []string `json:"interests"`
		Gender    string   `json:"gender"`
		Location  string   `json:"location"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	minAge := int32(18)
	if req.MinAge != nil {
		minAge = *req.MinAge
	}
	maxAge := int32(99)
	if req.MaxAge != nil {
		maxAge = *req.MaxAge
	}

	// Validate matching request fields
	if req.Age < 13 || req.Age > 150 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "age must be between 13 and 150"})
		return
	}
	if minAge > maxAge {
		c.JSON(http.StatusBadRequest, gin.H{"error": "min_age must not exceed max_age"})
		return
	}
	if len(req.Interests) > 30 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "interests limited to 30 items"})
		return
	}
	if len(req.Gender) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gender too long"})
		return
	}
	if len(req.Location) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "location too long"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := h.matchingClient.JoinQueue(ctx, &matchingpb.JoinQueueRequest{
		UserId:    userID,
		Age:       req.Age,
		MinAge:    minAge,
		MaxAge:    maxAge,
		Interests: req.Interests,
		Gender:    req.Gender,
		Location:  req.Location,
	})
	if err != nil {
		h.log.Error("start matching failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *MatchingHandler) StopMatching(c *gin.Context) {
	userID := c.GetString("userID")
	token := extractToken(c)

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := h.matchingClient.LeaveQueue(ctx, &matchingpb.LeaveQueueRequest{
		UserId: userID,
	})
	if err != nil {
		h.log.Error("stop matching failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *MatchingHandler) GetMatchingStatus(c *gin.Context) {
	userID := c.GetString("userID")
	token := extractToken(c)

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := h.matchingClient.GetMatchingStatus(ctx, &matchingpb.GetMatchingStatusRequest{
		UserId: userID,
	})
	if err != nil {
		h.log.Error("get matching status failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *MatchingHandler) GetMatchHistory(c *gin.Context) {
	userID := c.GetString("userID")
	token := extractToken(c)

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
	const maxOffset = 1_000_000
	if offset > maxOffset {
		offset = maxOffset
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := h.matchingClient.GetMatchHistory(ctx, &matchingpb.GetMatchHistoryRequest{
		UserId: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		h.log.Error("get match history failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *MatchingHandler) EndMatch(c *gin.Context) {
	userID := c.GetString("userID")
	token := extractToken(c)

	var req struct {
		MatchID string `json:"match_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := h.matchingClient.EndMatch(ctx, &matchingpb.EndMatchRequest{
		MatchId: req.MatchID,
		UserId:  userID,
	})
	if err != nil {
		h.log.Error("end match failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

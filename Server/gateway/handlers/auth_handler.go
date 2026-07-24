package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/yourusername/connect/api-gateway/circuitbreaker"
	"github.com/yourusername/connect/api-gateway/config"
	pb "github.com/yourusername/connect/proto/auth"
)

type AuthHandler struct {
	conn        *grpc.ClientConn
	client      pb.AuthServiceClient
	log         *zap.Logger
	grpcTimeout time.Duration
}

func NewAuthHandler(cfg *config.Config, log *zap.Logger, cbr *circuitbreaker.Registry) *AuthHandler {
	opts := append([]grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}, cbr.DialOptions("AuthService")...)

	conn, err := grpc.NewClient(cfg.AuthServiceAddr, opts...)
	if err != nil {
		log.Fatal("failed to connect to auth service", zap.Error(err))
	}

	return &AuthHandler{
		conn:        conn,
		client:      pb.NewAuthServiceClient(conn),
		log:         log.Named("auth"),
		grpcTimeout: cfg.GRPCTimeout,
	}
}

// Close releases the underlying gRPC connection.
func (h *AuthHandler) Close() error { return h.conn.Close() }

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	resp, err := h.client.Register(ctx, &pb.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.log.Error("register call failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	resp, err := h.client.Login(ctx, &pb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.log.Error("login call failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusUnauthorized, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	resp, err := h.client.RefreshToken(ctx, &pb.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		h.log.Error("refresh token call failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusUnauthorized, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// The auth middleware already validated the token and stored the userID.
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	uid, ok := userIDVal.(string)
	if !ok || uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	resp, err := h.client.Logout(ctx, &pb.LogoutRequest{
		UserId: uid,
		Token:  token,
	})
	if err != nil {
		h.log.Error("logout call failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Service unavailable"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ValidateToken is a helper used by the auth middleware.
func (h *AuthHandler) ValidateToken(ctx context.Context, token string) (string, string, bool) {
	resp, err := h.client.ValidateToken(ctx, &pb.ValidateTokenRequest{Token: token})
	if err != nil || !resp.Valid {
		return "", "", false
	}
	email := resp.Email
	if email == "" {
		// Fallback: extract email from already-validated JWT payload
		email = extractEmailFromJWT(token)
	}
	return resp.UserId, email, true
}

// extractEmailFromJWT reads the email claim from a JWT payload without
// signature verification (the auth-service already validated the token).
func extractEmailFromJWT(token string) string {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	return claims.Email
}

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"XTalk/gateway/config"
	"XTalk/gateway/utils"
	authpb "XTalk/proto/auth"
	userpb "XTalk/proto/user"
)

type UserHandler struct {
	userClient userpb.UserServiceClient
	authClient authpb.AuthServiceClient
	cfg        *config.Config
}

func NewUserHandler(cfg *config.Config) *UserHandler {
	userConn, _ := grpc.NewClient(
		"localhost:"+cfg.UserServicePort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	authConn, _ := grpc.NewClient(
		"localhost:"+cfg.AuthServicePort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	return &UserHandler{
		userClient: userpb.NewUserServiceClient(userConn),
		authClient: authpb.NewAuthServiceClient(authConn),
		cfg:        cfg,
	}
}

func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := utils.ExtractToken(r)
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Validate token
	validateResp, err := h.authClient.ValidateToken(ctx, &authpb.ValidateTokenRequest{Token: token})
	if err != nil || !validateResp.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Add token to metadata
	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Get user
	userResp, err := h.userClient.GetUser(ctx, &userpb.GetUserRequest{
		UserId: validateResp.UserId,
	})

	if err != nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if !userResp.Success {
		w.WriteHeader(http.StatusNotFound)
	}
	json.NewEncoder(w).Encode(userResp)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := utils.ExtractToken(r)
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Validate token
	validateResp, err := h.authClient.ValidateToken(ctx, &authpb.ValidateTokenRequest{Token: token})
	if err != nil || !validateResp.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	var req struct {
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx = metadata.NewOutgoingContext(ctx, md)

	userResp, err := h.userClient.UpdateUser(ctx, &userpb.UpdateUserRequest{
		UserId:    validateResp.UserId,
		Username:  req.Username,
		AvatarUrl: req.AvatarURL,
	})

	if err != nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userResp)
}

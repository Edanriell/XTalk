package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"XTalk/gateway/config"
	"XTalk/gateway/utils"
	pb "XTalk/proto/auth"
)

type AuthHandler struct {
	client pb.AuthServiceClient
	cfg    *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	conn, err := grpc.NewClient(
		"localhost:"+cfg.AuthServicePort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}

	return &AuthHandler{
		client: pb.NewAuthServiceClient(conn),
		cfg:    cfg,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.client.Register(ctx, &pb.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if !resp.Success {
		w.WriteHeader(http.StatusBadRequest)
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.client.Login(ctx, &pb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if !resp.Success {
		w.WriteHeader(http.StatusUnauthorized)
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.client.RefreshToken(ctx, &pb.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	})

	if err != nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if !resp.Success {
		w.WriteHeader(http.StatusUnauthorized)
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := utils.ExtractToken(r)
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Validate token first
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	validateResp, err := h.client.ValidateToken(ctx, &pb.ValidateTokenRequest{Token: token})
	if err != nil || !validateResp.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	resp, err := h.client.Logout(ctx, &pb.LogoutRequest{
		UserId: validateResp.UserId,
		Token:  token,
	})

	if err != nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

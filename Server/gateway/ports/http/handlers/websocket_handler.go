package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	"XTalk/gateway/application"
	authpb "XTalk/proto/auth"
	msgpb "XTalk/proto/message"
	userpb "XTalk/proto/user"
)

type WebSocketHandler struct {
	authClient     authpb.AuthServiceClient
	userClient     userpb.UserServiceClient
	msgClient      msgpb.MessageServiceClient
	log            *zap.Logger
	clients        map[string]*Client
	rooms          map[string]map[string]struct{} // roomID -> set of userIDs
	mu             sync.RWMutex
	allowedOrigins map[string]struct{}
	readBufSize    int
	writeBufSize   int
	grpcTimeout    time.Duration
}

type Client struct {
	UserID    string
	Conn      *websocket.Conn
	Send      chan []byte
	Rooms     map[string]struct{} // rooms this client is in
	closeOnce sync.Once           // ensures Send is closed exactly once
	closed    atomic.Bool         // true after closeSend(); prevents send-on-closed-channel
}

// closeSend safely closes the Send channel exactly once.
func (c *Client) closeSend() {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		close(c.Send)
	})
}

// trySend attempts to send on the Send channel. Returns false if the channel
// is closed or full.
func (c *Client) trySend(data []byte) bool {
	if c.closed.Load() {
		return false
	}
	select {
	case c.Send <- data:
		return true
	default:
		return false
	}
}

func NewWebSocketHandler(
	cfg *application.Config,
	log *zap.Logger,
	authClient authpb.AuthServiceClient,
	userClient userpb.UserServiceClient,
	messageClient msgpb.MessageServiceClient,
) *WebSocketHandler {
	// Parse allowed origins from config
	origins := make(map[string]struct{})
	for _, o := range cfg.AllowedOrigins {
		origins[strings.TrimSpace(o)] = struct{}{}
	}

	return &WebSocketHandler{
		authClient:     authClient,
		userClient:     userClient,
		msgClient:      messageClient,
		log:            log.Named("websocket"),
		clients:        make(map[string]*Client),
		rooms:          make(map[string]map[string]struct{}),
		allowedOrigins: origins,
		readBufSize:    cfg.WSReadBufferSize,
		writeBufSize:   cfg.WSWriteBufferSize,
		grpcTimeout:    cfg.GRPCTimeout,
	}
}

func (h *WebSocketHandler) upgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  h.readBufSize,
		WriteBufferSize: h.writeBufSize,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // same-origin or non-browser client
			}
			if len(h.allowedOrigins) == 0 {
				return false // no origins configured — reject cross-origin
			}
			_, ok := h.allowedOrigins[origin]
			return ok
		},
	}
}

func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.grpcTimeout)
	defer cancel()

	validateResp, err := h.authClient.ValidateToken(ctx, &authpb.ValidateTokenRequest{Token: token})
	if err != nil || !validateResp.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	up := h.upgrader()
	conn, err := up.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.log.Error("failed to upgrade connection", zap.Error(err))
		return
	}

	client := &Client{
		UserID: validateResp.UserId,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Rooms:  make(map[string]struct{}),
	}

	h.mu.Lock()
	// Evict existing connection for this user to prevent zombie goroutines.
	if old, exists := h.clients[validateResp.UserId]; exists {
		old.closeSend()  // signals writePump to exit
		old.Conn.Close() // unblocks readPump
	}
	h.clients[validateResp.UserId] = client
	h.mu.Unlock()

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	statusCtx, statusCancel := context.WithTimeout(context.Background(), h.grpcTimeout)
	defer statusCancel()
	statusCtx = metadata.NewOutgoingContext(statusCtx, md)
	h.userClient.UpdateStatus(statusCtx, &userpb.UpdateStatusRequest{
		UserId: validateResp.UserId,
		Status: "online",
	})

	h.log.Info("user connected via websocket", zap.String("userID", validateResp.UserId))

	go h.writePump(client)
	go h.readPump(client, token)
}

// JoinRoom adds a client to a room for targeted broadcasts
func (h *WebSocketHandler) JoinRoom(userID, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client, ok := h.clients[userID]; ok {
		client.Rooms[roomID] = struct{}{}
	}

	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[string]struct{})
	}
	h.rooms[roomID][userID] = struct{}{}
}

// LeaveRoom removes a client from a room
func (h *WebSocketHandler) LeaveRoom(userID, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client, ok := h.clients[userID]; ok {
		delete(client.Rooms, roomID)
	}

	if members, ok := h.rooms[roomID]; ok {
		delete(members, userID)
		if len(members) == 0 {
			delete(h.rooms, roomID)
		}
	}
}

func (h *WebSocketHandler) readPump(client *Client, token string) {
	defer func() {
		if r := recover(); r != nil {
			h.log.Error("panic in readPump", zap.String("userID", client.UserID), zap.Any("recover", r))
		}

		h.mu.Lock()
		// Remove client from all rooms
		for roomID := range client.Rooms {
			if members, ok := h.rooms[roomID]; ok {
				delete(members, client.UserID)
				if len(members) == 0 {
					delete(h.rooms, roomID)
				}
			}
		}
		delete(h.clients, client.UserID)
		h.mu.Unlock()

		client.closeSend() // signal writePump to exit before closing the connection
		client.Conn.Close()

		md := metadata.New(map[string]string{"authorization": "Bearer " + token})
		ctx, cancel := context.WithTimeout(context.Background(), h.grpcTimeout)
		defer cancel()
		ctx = metadata.NewOutgoingContext(ctx, md)
		h.userClient.UpdateStatus(ctx, &userpb.UpdateStatusRequest{
			UserId: client.UserID,
			Status: "offline",
		})

		h.log.Info("user disconnected", zap.String("userID", client.UserID))
	}()

	client.Conn.SetReadLimit(65536) // 64 KB max message size
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			break
		}
		h.handleMessage(client, message, token)
	}
}

func (h *WebSocketHandler) writePump(client *Client) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		if r := recover(); r != nil {
			h.log.Error("panic in writePump", zap.String("userID", client.UserID), zap.Any("recover", r))
		}
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *WebSocketHandler) handleMessage(client *Client, message []byte, token string) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		return
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "chat_message":
		h.handleChatMessage(client, msg, token)
	case "typing":
		h.handleTypingIndicator(client, msg)
	case "join_room":
		if roomID, ok := msg["room_id"].(string); ok && roomID != "" {
			h.JoinRoom(client.UserID, roomID)
		}
	case "leave_room":
		if roomID, ok := msg["room_id"].(string); ok && roomID != "" {
			h.LeaveRoom(client.UserID, roomID)
		}
	}
}

func (h *WebSocketHandler) handleChatMessage(client *Client, msg map[string]interface{}, token string) {
	roomID, _ := msg["room_id"].(string)
	content, _ := msg["content"].(string)

	if roomID == "" || content == "" {
		return
	}

	// Enforce content length limit before making the gRPC call.
	const maxContentLen = 10_000 // matches message-service MaxContentLength
	if len(content) > maxContentLen {
		return
	}

	// Verify the client has joined this room.
	h.mu.RLock()
	_, inRoom := client.Rooms[roomID]
	h.mu.RUnlock()
	if !inRoom {
		return
	}

	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := h.msgClient.SendMessage(ctx, &msgpb.SendMessageRequest{
		ChatId:      roomID,
		SenderId:    client.UserID,
		Content:     content,
		MessageType: "text",
	})
	if err != nil || !resp.Success {
		return
	}

	// Build a structured message payload for WebSocket clients.
	broadcast := map[string]interface{}{
		"type":         "chat_message",
		"message_id":   resp.MessageId,
		"chat_id":      roomID,
		"sender_id":    client.UserID,
		"content":      content,
		"message_type": "text",
		"timestamp":    time.Now().Unix(),
	}
	broadcastData, err := json.Marshal(broadcast)
	if err != nil {
		return
	}
	h.broadcastToRoom(roomID, broadcastData)
}

func (h *WebSocketHandler) handleTypingIndicator(client *Client, msg map[string]interface{}) {
	roomID, _ := msg["room_id"].(string)
	isTyping, _ := msg["is_typing"].(bool)

	if roomID == "" {
		return
	}

	notification := map[string]interface{}{
		"type":      "typing",
		"user_id":   client.UserID,
		"room_id":   roomID,
		"is_typing": isTyping,
		"timestamp": time.Now().Unix(),
	}

	data, err := json.Marshal(notification)
	if err != nil {
		h.log.Error("failed to marshal typing indicator", zap.Error(err))
		return
	}
	h.broadcastToRoom(roomID, data)
}

func (h *WebSocketHandler) broadcastToRoom(roomID string, data []byte) {

	// Collect slow clients under RLock; evict them afterwards under write Lock.
	var slowClients []string

	h.mu.RLock()
	members, ok := h.rooms[roomID]
	if !ok {
		h.mu.RUnlock()
		return
	}

	for userID := range members {
		client, exists := h.clients[userID]
		if !exists {
			continue
		}
		if !client.trySend(data) {
			h.log.Warn("client send buffer full, disconnecting slow client", zap.String("userID", userID))
			client.closeSend()
			slowClients = append(slowClients, userID)
		}
	}
	h.mu.RUnlock()

	if len(slowClients) > 0 {
		h.mu.Lock()
		for _, uid := range slowClients {
			delete(h.clients, uid)
			// Also remove from room membership to prevent stale lookups.
			if members, ok := h.rooms[roomID]; ok {
				delete(members, uid)
				if len(members) == 0 {
					delete(h.rooms, roomID)
				}
			}
		}
		h.mu.Unlock()
	}
}

// SendToUser sends a payload to a specific connected user via WebSocket.
func (h *WebSocketHandler) SendToUser(userID string, payload []byte) {
	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	if !client.trySend(payload) {
		h.log.Warn("SendToUser: client buffer full, disconnecting", zap.String("userID", userID))
		client.closeSend()
		h.mu.Lock()
		delete(h.clients, userID)
		h.mu.Unlock()
	}
}

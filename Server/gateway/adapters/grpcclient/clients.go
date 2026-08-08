package grpcclient

import (
	"XTalk/gateway/adapters/circuitbreaker"
	"XTalk/gateway/application"
	authpb "XTalk/proto/auth"
	chatpb "XTalk/proto/chat"
	matchingpb "XTalk/proto/matching"
	messagepb "XTalk/proto/message"
	userpb "XTalk/proto/user"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Clients struct {
	Auth     authpb.AuthServiceClient
	User     userpb.UserServiceClient
	Chat     chatpb.ChatServiceClient
	Message  messagepb.MessageServiceClient
	Matching matchingpb.MatchingServiceClient
	conns    []*grpc.ClientConn
}

func New(cfg *application.Config, breakers *circuitbreaker.Registry) (*Clients, error) {
	clients := &Clients{}

	authConn, err := dial(cfg.AuthServiceAddr, breakers.DialOptions("AuthService"))
	if err != nil {
		return nil, fmt.Errorf("connect to auth service: %w", err)
	}
	clients.conns = append(clients.conns, authConn)
	clients.Auth = authpb.NewAuthServiceClient(authConn)

	userConn, err := dial(cfg.UserServiceAddr, breakers.DialOptions("UserService"))
	if err != nil {
		_ = clients.Close()
		return nil, fmt.Errorf("connect to user service: %w", err)
	}
	clients.conns = append(clients.conns, userConn)
	clients.User = userpb.NewUserServiceClient(userConn)

	chatConn, err := dial(cfg.ChatServiceAddr, breakers.DialOptions("ChatService"))
	if err != nil {
		_ = clients.Close()
		return nil, fmt.Errorf("connect to chat service: %w", err)
	}
	clients.conns = append(clients.conns, chatConn)
	clients.Chat = chatpb.NewChatServiceClient(chatConn)

	messageConn, err := dial(cfg.MessageServiceAddr, breakers.DialOptions("MessageService"))
	if err != nil {
		_ = clients.Close()
		return nil, fmt.Errorf("connect to message service: %w", err)
	}
	clients.conns = append(clients.conns, messageConn)
	clients.Message = messagepb.NewMessageServiceClient(messageConn)

	matchingConn, err := dial(cfg.MatchingServiceAddr, breakers.DialOptions("MatchingService"))
	if err != nil {
		_ = clients.Close()
		return nil, fmt.Errorf("connect to matching service: %w", err)
	}
	clients.conns = append(clients.conns, matchingConn)
	clients.Matching = matchingpb.NewMatchingServiceClient(matchingConn)

	return clients, nil
}

func dial(address string, options []grpc.DialOption) (*grpc.ClientConn, error) {
	base := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	return grpc.NewClient(address, append(base, options...)...)
}

func (c *Clients) Close() error {
	var first error
	for _, conn := range c.conns {
		if err := conn.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

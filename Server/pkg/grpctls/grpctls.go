package grpctls

import (
	"crypto/tls"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// ServerOptions returns gRPC server credentials for TLS.
// If certFile or keyFile is empty, insecure (plaintext) is used.
func ServerOptions(certFile, keyFile string) (grpc.ServerOption, error) {
	if certFile == "" || keyFile == "" {
		return nil, nil // caller should skip adding the option
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load server cert/key: %w", err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	return grpc.Creds(credentials.NewTLS(tlsCfg)), nil
}

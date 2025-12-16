package grpctls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
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

// MutualTLSServerOptions returns gRPC server credentials with mutual TLS.
// Clients must present a certificate signed by the given CA.
func MutualTLSServerOptions(certFile, keyFile, caFile string) (grpc.ServerOption, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load server cert/key: %w", err)
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}

	return grpc.Creds(credentials.NewTLS(tlsCfg)), nil
}

// ClientDialOption returns a gRPC dial option for TLS or insecure connections.
// If caFile is empty, insecure credentials are used.
func ClientDialOption(caFile string) (grpc.DialOption, error) {
	if caFile == "" {
		return grpc.WithTransportCredentials(insecure.NewCredentials()), nil
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsCfg := &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	}

	return grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)), nil
}

// MutualTLSClientDialOption returns a gRPC dial option for mutual TLS.
func MutualTLSClientDialOption(certFile, keyFile, caFile string) (grpc.DialOption, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load client cert/key: %w", err)
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS12,
	}

	return grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)), nil
}

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"github.com/mhsantos/rlcp/cmd/internal/pb"
	"github.com/mhsantos/rlcp/cmd/server/internal/storage"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	slog.Debug("starting server")
	// Listen for incoming connections on port 8080
	ln, err := net.Listen("tcp", ":8087")
	if err != nil {
		slog.Error("error listening", slog.Any("error", err))
		return
	}

	// db
	storage := storage.NewMemStorage()

	tlsConfig, err := getTLSConfig()
	if err != nil {
		slog.Error("exiting due to TLS config error", slog.Any("error", err))
		os.Exit(1)
	}

	s := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	server := NewServer(storage)
	pb.RegisterRemoteExecutorServer(s, server)

	if err := s.Serve(ln); err != nil {
		slog.Error("failed to serve grpc server", slog.Any("error", err))
	}
}

func getTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair("x509/server_cert.pem", "x509/server_key.pem")
	if err != nil {
		slog.Error("failed to load key pair", slog.Any("error", err))
		return nil, err
	}

	ca := x509.NewCertPool()
	caFilePath := "x509/ca_cert.pem"
	caBytes, err := os.ReadFile(caFilePath)
	if err != nil {
		slog.Error("failed to read ca cert", slog.String("path", caFilePath), slog.Any("error", err))
		return nil, err
	}
	if ok := ca.AppendCertsFromPEM(caBytes); !ok {
		slog.Error("failed to parse", slog.String("path", caFilePath))
		return nil, err
	}

	return &tls.Config{
		ClientAuth:            tls.RequireAndVerifyClientCert,
		Certificates:          []tls.Certificate{cert},
		ClientCAs:             ca,
		VerifyPeerCertificate: authorizeClient,
		MinVersion:            tls.VersionTLS13,
		MaxVersion:            tls.VersionTLS13,
	}, nil
}

func authorizeClient(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	// Check if any certificates were provided
	if len(rawCerts) == 0 || len(verifiedChains) == 0 {
		return fmt.Errorf("no client certificate provided")
	}

	// Extract the client certificate
	clientCert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("failed to parse client certificate: %w", err)
	}

	// Check the Common Name
	expectedCN := "marcel+client@email.com" // Replace with your expected CN
	if clientCert.Subject.CommonName != expectedCN {
		return fmt.Errorf("invalid client certificate CN: got %s, want %s", clientCert.Subject.CommonName, expectedCN)
	}

	fmt.Println("Client certificate CN verified:", clientCert.Subject.CommonName)
	return nil
}

func getRequesterEmail(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	authInfo := p.AuthInfo.(credentials.TLSInfo)
	return authInfo.State.VerifiedChains[0][0].Subject.CommonName
}

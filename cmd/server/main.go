package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"github.com/mhsantos/rlcp/cmd/internal/pb"
	"github.com/mhsantos/rlcp/cmd/server/internal/storage"
)

func main() {
	addr := flag.String("addr", ":8087", "server listen address")
	certFile := flag.String("cert", "x509/server_cert.pem", "path to server certificate")
	keyFile := flag.String("key", "x509/server_key.pem", "path to server private key")
	caFile := flag.String("ca", "x509/ca_cert.pem", "path to CA certificate")
	flag.Parse()

	slog.SetLogLoggerLevel(slog.LevelDebug)
	slog.Debug("starting server")

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		slog.Error("error listening", slog.Any("error", err))
		os.Exit(1)
	}

	db := storage.NewMemStorage()

	tlsConfig, err := getTLSConfig(*certFile, *keyFile, *caFile)
	if err != nil {
		slog.Error("exiting due to TLS config error", slog.Any("error", err))
		os.Exit(1)
	}

	s := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	srv := NewServer(db)
	pb.RegisterRemoteExecutorServer(s, srv)

	// Graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		slog.Info("shutting down server gracefully")
		s.GracefulStop()
	}()

	slog.Info("server listening", slog.String("addr", *addr))
	if err := s.Serve(ln); err != nil {
		slog.Error("failed to serve grpc server", slog.Any("error", err))
	}
}

func getTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		slog.Error("failed to load key pair", slog.Any("error", err))
		return nil, err
	}

	ca := x509.NewCertPool()
	caBytes, err := os.ReadFile(caFile)
	if err != nil {
		slog.Error("failed to read ca cert", slog.String("path", caFile), slog.Any("error", err))
		return nil, err
	}
	if ok := ca.AppendCertsFromPEM(caBytes); !ok {
		return nil, fmt.Errorf("failed to parse CA certificate: %s", caFile)
	}

	return &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    ca,
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
	}, nil
}

func getRequesterEmail(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return ""
	}
	if len(tlsInfo.State.VerifiedChains) == 0 || len(tlsInfo.State.VerifiedChains[0]) == 0 {
		return ""
	}
	return tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
}

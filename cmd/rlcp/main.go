package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/mhsantos/rlcp/cmd/cli"
	"github.com/mhsantos/rlcp/cmd/internal/pb"
)

func main() {
	serverAddr := flag.String("addr", "localhost:8087", "server address")
	certFile := flag.String("cert", "x509/client_cert.pem", "path to client certificate")
	keyFile := flag.String("key", "x509/client_key.pem", "path to client private key")
	caFile := flag.String("ca", "x509/ca_cert.pem", "path to CA certificate")
	serverName := flag.String("server-name", "localhost", "expected server certificate CN")
	flag.Parse()

	slog.SetLogLoggerLevel(slog.LevelDebug)
	slog.Debug("starting client")

	args := flag.Args()
	// Prepend program name for cli.ParseCommand compatibility
	args = append([]string{os.Args[0]}, args...)

	option, err := cli.ParseCommand(args)
	if err != nil {
		fmt.Println(err)
		fmt.Println(cli.HelpPrompt)
		os.Exit(1)
	}

	sendCommandToServer(option, *serverAddr, *certFile, *keyFile, *caFile, *serverName)
}

func getTLSConfig(certFile, keyFile, caFile, serverName string) (*tls.Config, error) {
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
		ServerName:   serverName,
		Certificates: []tls.Certificate{cert},
		RootCAs:      ca,
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
	}, nil
}

func sendCommandToServer(option cli.Option, serverAddr, certFile, keyFile, caFile, serverName string) {
	tlsConfig, err := getTLSConfig(certFile, keyFile, caFile, serverName)
	if err != nil {
		slog.Error("exiting due to TLS config error", slog.Any("error", err))
		os.Exit(1)
	}

	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		slog.Error("error creating GRPC client", slog.Any("error", err))
		return
	}
	defer conn.Close()

	client := pb.NewRemoteExecutorClient(conn)

	switch option.Op {
	case cli.Run:
		jobId, err := callRunCommand(client, option.Args)
		if err != nil {
			slog.Error("error scheduling command", slog.Any("error", err))
			return
		}
		fmt.Printf("Job ID: %s\n", jobId)
	case cli.Status:
		status, err := callGetStatus(client, option.Args[0])
		if err != nil {
			slog.Error("error getting status", slog.Any("error", err))
			return
		}
		fmt.Printf("Job Status: %s\n", status)
	case cli.Output:
		err := callGetOutput(client, option.Args[0])
		if err != nil {
			slog.Error("error getting output", slog.Any("error", err))
			return
		}
	case cli.Stop:
		status, err := callStop(client, option.Args[0])
		if err != nil {
			slog.Error("error stopping command", slog.Any("error", err))
			return
		}
		fmt.Printf("Job Status: %s\n", status)
	case cli.Help:
		fmt.Println(cli.HelpPrompt)
	default:
		slog.Error("invalid operation", slog.Any("op", option.Op))
	}
}

func callRunCommand(client pb.RemoteExecutorClient, args []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var cmdArgs []string
	if len(args) > 1 {
		cmdArgs = args[1:]
	}

	req := &pb.CmdRequest{
		Command:   args[0],
		Arguments: cmdArgs,
	}
	resp, err := client.ExecCommand(ctx, req)
	if err != nil {
		slog.Error("error calling server", slog.Any("error", err))
		return "", err
	}
	return resp.JobId, nil
}

func callGetOutput(client pb.RemoteExecutorClient, jobId string) error {
	// Use a cancellable context so Ctrl+C cleanly terminates the stream
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	stream, err := client.GetOutput(ctx, &pb.GetRequest{JobId: jobId})
	if err != nil {
		slog.Error("call to client.GetOutput failed", slog.Any("error", err))
		return err
	}
	for {
		output, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			slog.Error("client.GetOutput stream iteration failed", slog.Any("error", err))
			return err
		}
		fmt.Print(string(output.Output))
	}
}

func callGetStatus(client pb.RemoteExecutorClient, jobId string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	status, err := client.GetStatus(ctx, &pb.GetRequest{JobId: jobId})
	if err != nil {
		slog.Error("call to client.GetStatus failed", slog.Any("error", err))
		return "", err
	}
	return status.Status.String(), nil
}

func callStop(client pb.RemoteExecutorClient, jobId string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := client.StopJob(ctx, &pb.StopRequest{JobId: jobId})
	if err != nil {
		slog.Error("call stopping process", slog.Any("error", err))
		return "", err
	}
	status, err := client.GetStatus(ctx, &pb.GetRequest{JobId: jobId})
	if err != nil {
		slog.Error("call to client.GetStatus failed", slog.Any("error", err))
		return "", err
	}
	return status.Status.String(), nil
}

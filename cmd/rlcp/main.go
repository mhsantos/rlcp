package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/mhsantos/rlcp/cmd/cli"
	"github.com/mhsantos/rlcp/cmd/internal/pb"
)

type Command struct {
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	slog.Debug("starting client")
	args := os.Args
	for idx, arg := range args {
		slog.Debug("argument", slog.Int("position", idx), slog.String("value", arg))
	}

	option, err := cli.ParseCommand(args)
	if err != nil {
		fmt.Println(err)
		fmt.Println(cli.HelpPrompt)
		os.Exit(1)
	}

	sendCommandToServer(option)
}

func getTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair("x509/client_cert.pem", "x509/client_key.pem")
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
		ServerName:   "localhost",
		Certificates: []tls.Certificate{cert},
		RootCAs:      ca,
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
	}, nil
}

func sendCommandToServer(option cli.Option) {
	tlsConfig, err := getTLSConfig()
	if err != nil {
		slog.Error("exiting due to TLS config error", slog.Any("error", err))
		os.Exit(1)
	}

	// Connect to the server
	conn, err := grpc.NewClient("localhost:8087", grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
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
		err := callGetResponse(client, option.Args[0])
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
	default:
		slog.Error("invalid operation", slog.String("op", string(option.Op)))
	}
}

func callRunCommand(client pb.RemoteExecutorClient, args []string) (string, error) {
	ctx := context.Background()

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

func callGetResponse(client pb.RemoteExecutorClient, jobId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.GetOutput(ctx, &pb.GetRequest{JobId: jobId})
	if err != nil {
		slog.Error("call to client.GetResult failed", slog.Any("error", err))
		return err
	}
	for {
		output, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			slog.Error("client.GetResult stream iteration failed", slog.Any("error", err))
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

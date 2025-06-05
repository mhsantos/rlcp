package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mhsantos/rlcp/cmd/internal/pb"
	"github.com/mhsantos/rlcp/cmd/server/internal/executor"
	"github.com/mhsantos/rlcp/cmd/server/internal/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type server struct {
	pb.UnimplementedRemoteExecutorServer
	db storage.JobStorage
}

func NewServer(db storage.JobStorage) *server {
	return &server{
		db: db,
	}
}
func (s *server) ExecCommand(ctx context.Context, req *pb.CmdRequest) (*pb.JobDetails, error) {
	email := getRequesterEmail(ctx)
	if len(email) == 0 {
		slog.Error("invalid client email")
		return nil, errors.New("email not informed on CommonName")
	}

	userId, ok := s.db.GetUserId(email)
	if !ok {
		slog.Error("user id not found")
		return nil, errors.New("couldn't find a user for the informed email")
	}

	if !s.db.Authorized(userId, storage.Run) {
		slog.Error("not authorized")
		return nil, errors.New("user not authorized to Run commands")
	}

	job := storage.NewJob()
	s.db.SaveJob(job.Id.String(), job)

	command := req.Command
	args := req.Arguments

	// Print the incoming data
	slog.Debug("Received", slog.String("value", command))

	err := executor.RunCommand(job, command, args)
	if err != nil {
		slog.Error("error calling command execution")
		return nil, err
	}

	return &pb.JobDetails{
		JobId:  job.Id.String(),
		Status: pb.JobDetails_RUNNING,
	}, nil
}

func (s *server) GetStatus(ctx context.Context, req *pb.GetRequest) (*pb.JobDetails, error) {
	jobId := req.JobId
	job, ok := s.db.GetJob(jobId)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "Could not find a job for the id provided")
	}

	fmt.Printf("jobid: %s, status:%s, pbStatus: %s\n", jobId, job.Status, pb.JobDetails_Status(job.Status))

	return &pb.JobDetails{
		JobId:  jobId,
		Status: pb.JobDetails_Status(job.Status),
	}, nil
}

func (s *server) GetOutput(req *pb.GetRequest, stream grpc.ServerStreamingServer[pb.JobOutput]) error {
	jobId := req.JobId
	job, ok := s.db.GetJob(jobId)
	if !ok {
		return status.Errorf(codes.NotFound, "Could not find a job for the id provided")
	}

	outCh := make(chan []byte)

	go job.RegisterListener(outCh)

	for out := range outCh {
		err := stream.Send(&pb.JobOutput{Output: out})
		if err != nil {
			slog.Error("error sending response to client", slog.Any("error", err))
			return err
		}
	}
	return nil
}

func (s *server) StopJob(ctx context.Context, req *pb.StopRequest) (*emptypb.Empty, error) {
	jobId := req.JobId

	job, ok := s.db.GetJob(jobId)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "Could not find a job for the id provided")
	}

	if err := job.Cmd.Process.Kill(); err != nil {
		job.Status = storage.Errored
		return nil, status.Errorf(codes.Unknown, "Error killing the process: %v", err)
	}
	job.Status = storage.Stopped
	job.CloseListeners()

	return nil, nil
}

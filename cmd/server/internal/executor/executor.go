package executor

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"os/exec"

	"github.com/mhsantos/rlcp/cmd/server/internal/storage"
)

func RunCommand(ctx context.Context, job *storage.Job, command string, args []string) error {
	cmd := exec.CommandContext(ctx, command, args...)

	job.Cmd = cmd

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("error acquiring stdout pipe", slog.Any("error", err))
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		slog.Error("error acquiring stderr pipe", slog.Any("error", err))
		return err
	}

	err = cmd.Start()
	if err != nil {
		slog.Error("error starting command", slog.Any("error", err))
		return err
	}

	go ListenToCommandOutput(job, stdout, stderr)
	go waitCommand(cmd)

	return nil
}

// ListenToCommandOutput reads the output from stdout and stderr and sends it to
// LogHandler.ProcessOutput, which is responsible for storing it and forwarding it to listeners.
func ListenToCommandOutput(job *storage.Job, stdout, stderr io.ReadCloser) {
	defer stdout.Close()
	defer stderr.Close()

	stream := io.MultiReader(stdout, stderr)

	streamReader := bufio.NewReader(stream)
	buff := make([]byte, 1024)
	for {
		n, err := streamReader.Read(buff)
		if n > 0 {
			job.ProcessOutput(buff[:n])
		} else {
			if err != nil {
				slog.Error("finished reading?", slog.Any("err", err), slog.Bool("io.EOF", err == io.EOF))
				if err == io.EOF {
					job.Status = storage.Completed
					return
				}
			}
		}
	}
}

// waitCommand calls exec.Cmd.Wait(), which is required to start processing the command
func waitCommand(cmd *exec.Cmd) {
	_ = cmd.Wait()
}

package executor

import (
	"io"
	"log/slog"
	"os/exec"

	"github.com/mhsantos/rlcp/cmd/server/internal/storage"
)

func RunCommand(job *storage.Job, command string, args []string) error {
	cmd := exec.Command(command, args...)

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

	slog.Debug("listening to command output")

	stream := io.MultiReader(stdout, stderr)

	buff := make([]byte, 1024)
	for {
		n, err := stream.Read(buff)
		if n > 0 {
			err = job.ProcessOutput(buff[:n])
			if err != nil {
				slog.Error("error processing output", slog.Any("error", err))
				if err := job.Cmd.Process.Kill(); err != nil {
					slog.Error("error killing process", slog.Any("error", err))
					return
				}
			}
		} else {
			if err != nil {
				slog.Error("finished reading?", slog.Any("err", err), slog.Bool("io.EOF", err == io.EOF))
				if err == io.EOF {
					slog.Debug("ListenToCommandOutput EOF")
					job.Status = storage.Completed
					return
				}
				job.Status = storage.Errored
				return
			}
		}
	}
}

// waitCommand calls exec.Cmd.Wait(), which is required to start processing the command
func waitCommand(cmd *exec.Cmd) {
	_ = cmd.Wait()
}

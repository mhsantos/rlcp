package executor

import (
	"io"
	"log/slog"
	"os/exec"
	"sync"
	"syscall"

	"github.com/mhsantos/rlcp/cmd/server/internal/storage"
)

func RunCommand(job *storage.Job, command string, args []string) error {
	cmd := exec.Command(command, args...)

	// Create a new process group so we can kill the entire tree on stop
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

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

	go listenToCommandOutput(job, stdout, stderr)

	return nil
}

// KillProcessGroup sends SIGKILL to the entire process group, ensuring
// all child processes are terminated, not just the direct child.
func KillProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		return cmd.Process.Kill()
	}
	return syscall.Kill(-pgid, syscall.SIGKILL)
}

// listenToCommandOutput reads stdout and stderr concurrently and sends
// output to job.ProcessOutput. When both streams are exhausted, it waits
// for the process to exit and sets the terminal job status.
func listenToCommandOutput(job *storage.Job, stdout, stderr io.ReadCloser) {
	var wg sync.WaitGroup
	wg.Add(2)

	readStream := func(r io.ReadCloser) {
		defer wg.Done()
		defer r.Close()
		buf := make([]byte, 1024)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				if pErr := job.ProcessOutput(buf[:n]); pErr != nil {
					slog.Error("error processing output", slog.Any("error", pErr))
					KillProcessGroup(job.Cmd)
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					slog.Error("error reading command output", slog.Any("error", err))
				}
				return
			}
		}
	}

	go readStream(stdout)
	go readStream(stderr)

	wg.Wait()

	// Wait for the process to fully exit and capture the exit status
	err := job.Cmd.Wait()
	if err != nil {
		slog.Debug("command exited with error", slog.Any("error", err))
		// Only set Errored if the job wasn't explicitly stopped
		if job.GetStatus() != storage.Stopped {
			job.SetStatus(storage.Errored)
		}
		return
	}
	job.SetStatus(storage.Completed)
}

package storage

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/google/uuid"
)

type Operation uint
type JobStatus uint

const (
	logFileSize int = 1024 * 1024 // 1MB
)

const (
	Run Operation = iota
	Status
	Output
	Stop
)

const (
	Running JobStatus = iota
	Completed
	Errored
	Stopped
)

// JobStorage defines the methods persist and access job relevant data.
type JobStorage interface {
	// GetUserId returns the UUID for the user matching the identifier on the request
	GetUserId(email string) (string, bool)

	// Authorized validates if the user requesting an operation on a job is the same that scheduled it
	Authorized(userId string, op Operation) bool

	// SaveJob adds a job to the storage map, allowing it to be searched by key
	SaveJob(jobId string, job *Job)

	// Returns the details for a job, including its storage and output
	GetJob(jobId string) (*Job, bool)
}

// Job contains the fields necessary to identify a command running on the server
// and report its output to the clients
type Job struct {
	Id        uuid.UUID
	Cmd       *exec.Cmd
	mu        sync.Mutex
	status    JobStatus
	log       *CmdLog
	done      chan struct{} // closed when the job reaches a terminal state
	closeOnce sync.Once
	listeners []chan []byte
}

// CmdLog manages the files and byte buffers storing the output from a command
type CmdLog struct {
	nFiles int
	buffer *[]byte
}

func NewJob() *Job {
	buffer := make([]byte, 0)
	return &Job{
		Id: uuid.New(),
		log: &CmdLog{
			buffer: &buffer,
		},
		done:      make(chan struct{}),
		listeners: make([]chan []byte, 0),
	}
}

// SetStatus sets the job status in a thread-safe manner.
// If the status is terminal (Completed, Errored, Stopped), it also closes
// all listener channels and signals the done channel.
func (j *Job) SetStatus(s JobStatus) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.status = s
	if s == Completed || s == Errored || s == Stopped {
		j.closeOnce.Do(func() {
			for _, listener := range j.listeners {
				close(listener)
			}
			j.listeners = nil
			close(j.done)
		})
	}
}

// GetStatus returns the job status in a thread-safe manner.
func (j *Job) GetStatus() JobStatus {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.status
}

// Done returns a channel that is closed when the job reaches a terminal state.
func (j *Job) Done() <-chan struct{} {
	return j.done
}

// ProcessOutput receives an array of bytes from the command's output and sends it to the receiver channels.
// After that it stores it in a temporary buffer. Once that buffer is full, it's stored on disk and flushed.
func (j *Job) ProcessOutput(out []byte) error {
	j.mu.Lock()

	// Copy listeners under lock, then send outside lock to avoid deadlock
	listeners := make([]chan []byte, len(j.listeners))
	copy(listeners, j.listeners)

	j.log.appendBytes(out)

	if len(*j.log.buffer) >= logFileSize {
		err := persistLog(j)
		if err != nil {
			j.status = Errored
			j.mu.Unlock()
			return err
		}
		j.log.nFiles++
		*j.log.buffer = make([]byte, 0)
	}
	j.mu.Unlock()

	// Send to listeners outside the lock to prevent deadlock.
	// Each listener gets its own copy to prevent slice aliasing.
	for _, listener := range listeners {
		data := make([]byte, len(out))
		copy(data, out)
		select {
		case listener <- data:
		case <-j.done:
			return nil
		}
	}

	return nil
}

// RegisterListener adds a listener channel to the pool of listener channels.
// It also reads the files and buffer array to send all the output, from its beginning, to the client.
func (j *Job) RegisterListener(listener chan []byte) {
	// first read the logs stored in files
	i := 0
	for {
		j.mu.Lock()
		if i == j.log.nFiles {
			j.listeners = append(j.listeners, listener)
			// then loads the logs from the buffer
			readLogBuffer(listener, *j.log.buffer)
			if j.status != Running {
				// this is the scenario where we are registering a listener for a command that already
				// finished, no matter if due to error, complete or stop
				close(listener)
			}
			j.mu.Unlock()
			break
		}
		j.mu.Unlock()
		for i < j.log.nFiles {
			err := readFile(listener, fmt.Sprintf("%s_%d.log", j.Id, i))
			if err != nil {
				return
			}
			i++
		}
	}

	// Block until job is done
	<-j.done
}

// CloseListeners iterates over the listeners pool closing each listener channel.
// It is important in several scenarios where we have listeners getting output and a command ends,
// to do not leave those listeners hanging.
func (j *Job) CloseListeners() {
	j.mu.Lock()
	for _, listener := range j.listeners {
		close(listener)
	}
	j.mu.Unlock()
}

func (s JobStatus) String() string {
	switch s {
	case Running:
		return "Running"
	case Completed:
		return "Completed"
	case Errored:
		return "Errored"
	case Stopped:
		return "Stopped"
	default:
		return "Undefined"
	}
}

// appendBytes appends bytes to the output buffer
func (c *CmdLog) appendBytes(out []byte) {
	*c.buffer = append(*c.buffer, out...)
}

// readFile reads the contents of a log file from persistent storage
func readFile(ch chan []byte, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	buffer := make([]byte, 1024)
	for {
		bytesRead, err := file.Read(buffer)
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		data := make([]byte, bytesRead)
		copy(data, buffer[:bytesRead])
		ch <- data
	}
	return nil
}

func readLogBuffer(ch chan []byte, log []byte) {
	reader := bytes.NewReader(log)
	buffer := make([]byte, 1024)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buffer[:n])
			ch <- data
		}
		if err != nil {
			break
		}
	}
}

// persistLog writes the log buffer to persistent storage in a file named jobid_index.log
func persistLog(job *Job) error {
	filename := fmt.Sprintf("%s_%d.log", job.Id, job.log.nFiles)
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(*job.log.buffer)
	return err
}

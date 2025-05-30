package storage

import (
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
	Status    JobStatus
	Cmd       *exec.Cmd
	mu        sync.Mutex
	log       CmdLog
	listeners []chan []byte
}

// CmdLog manages the files and byte buffers storing the output from a command
type CmdLog struct {
	nFiles int
	buffer []byte
}

func NewJob() *Job {
	return &Job{
		Id: uuid.New(),
		log: CmdLog{
			buffer: make([]byte, 0),
		},
		listeners: make([]chan []byte, 0),
	}
}

// ProcessOutput receives an array of bytes from the command's output and sends it to the receiver channels.
// After that it stores it in a temporary buffer. Once that buffer is full, it's stored on disk and flushed.
func (j *Job) ProcessOutput(out []byte) {
	j.mu.Lock()
	for _, listener := range j.listeners {
		listener <- out
	}
	j.log.appendBytes(out)
	j.mu.Unlock()
}

// RegisterListener adds a listener channel to the poll of listener channels.
// It also reads the files and buffer array to send all the output, from its beginning to the client.
func (j *Job) RegisterListener(listener chan []byte) {
	i := 0
	for {
		j.mu.Lock()
		if i == j.log.nFiles {
			j.listeners = append(j.listeners, listener)
			listener <- j.log.buffer
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
	(*c).buffer = append(c.buffer, out...)
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
		ch <- buffer[:bytesRead]
	}
	return nil
}

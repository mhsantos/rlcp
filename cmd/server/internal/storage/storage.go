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
	Status    JobStatus
	Cmd       *exec.Cmd
	mu        sync.Mutex
	log       *CmdLog
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
		listeners: make([]chan []byte, 0),
	}
}

// ProcessOutput receives an array of bytes from the command's output and sends it to the receiver channels.
// After that it stores it in a temporary buffer. Once that buffer is full, it's stored on disk and flushed.
func (j *Job) ProcessOutput(out []byte) error {
	j.mu.Lock()
	for _, listener := range j.listeners {
		listener <- out
	}
	j.log.appendBytes(out)

	if len(*j.log.buffer) >= logFileSize {
		err := persistLog(j)
		if err != nil {
			j.Status = Errored
			j.mu.Unlock()
			return err
		}
		j.log.nFiles++
		*j.log.buffer = make([]byte, 0)
	}
	j.mu.Unlock()
	return nil
}

// RegisterListener adds a listener channel to the poll of listener channels.
// It also reads the files and buffer array to send all the output, from its beginning to the client.
func (j *Job) RegisterListener(listener chan []byte) {
	// first read the logs stored in files
	i := 0
	for {
		j.mu.Lock()
		if i == j.log.nFiles {
			j.listeners = append(j.listeners, listener)

			// then loads the logs from the buffer
			readLogBuffer(listener, *j.log.buffer)
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
	*c.buffer = append(*c.buffer, out...)
	//	(*c).buffer = append(c.buffer, out...)
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

func readLogBuffer(ch chan []byte, log []byte) {
	reader := bytes.NewReader(log)
	buffer := make([]byte, 1024) // Buffer to read into
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			ch <- buffer[:n]
		} else {
			if err != nil {
				if err == io.EOF {
					break
				}
			}
		}
	}
}

// persistLog writes the log buffer to persistent storage in a file named jobid_index.log
func persistLog(job *Job) error {
	filename := fmt.Sprintf("%s_%d.log", job.Id, job.log.nFiles)
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(*job.log.buffer)
	if err != nil {
		return err
	}
	return nil
}

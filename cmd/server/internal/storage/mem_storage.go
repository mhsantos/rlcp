package storage

import (
	"log/slog"

	"github.com/google/uuid"
)

type Permission uint

const (
	Read Permission = iota
	Write
)

type User struct {
	id    string
	email string
	role  Permission
}

type MemStorage struct {
	users map[string]User
	jobs  map[string]*Job
}

func NewMemStorage() JobStorage {
	s := &MemStorage{
		users: make(map[string]User),
		jobs:  make(map[string]*Job),
	}
	s.init()
	return s
}

func (m *MemStorage) GetUserId(email string) (string, bool) {
	for _, usr := range m.users {
		if usr.email == email {
			return usr.id, true
		}
	}
	return "", false
}

func (m *MemStorage) Authorized(userId string, op Operation) bool {
	slog.Debug("authorizing", slog.String("userid", userId), slog.Any("operation", op))
	usr, ok := m.users[userId]
	if !ok {
		return false
	}
	slog.Debug("authorizing", slog.Any("user role", usr.role))
	if (op == Run || op == Stop) && usr.role == Read {
		return false
	}
	return true
}

func (m *MemStorage) SaveJob(jobId string, job *Job) {
	m.jobs[jobId] = job
}

func (m *MemStorage) GetJob(jobId string) (*Job, bool) {
	job, ok := m.jobs[jobId]
	return job, ok
}

// init is a temporary method to populate the database with test data
// TODO: remove this and add methods to insert/remove users and test data
func (m *MemStorage) init() {
	userId := uuid.NewString()
	m.users[userId] = User{
		id:    userId,
		email: "marcel+client@email.com",
		role:  Write,
	}

	userId2 := uuid.NewString()
	m.users[userId2] = User{
		id:    userId2,
		email: "marcel+client2@email.com",
		role:  Read,
	}
}

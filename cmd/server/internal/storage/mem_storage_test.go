package storage

import "testing"

func TestGetUserId(t *testing.T) {
	s := NewMemStorage()

	tests := []struct {
		name      string
		email     string
		wantFound bool
	}{
		{"known write user", "marcel+client@email.com", true},
		{"known read user", "marcel+client2@email.com", true},
		{"unknown user", "unknown@email.com", false},
		{"empty email", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, found := s.GetUserId(tt.email)
			if found != tt.wantFound {
				t.Errorf("GetUserId(%q) found = %v, want %v", tt.email, found, tt.wantFound)
			}
			if found && id == "" {
				t.Errorf("GetUserId(%q) returned empty id for found user", tt.email)
			}
		})
	}
}

func TestAuthorized(t *testing.T) {
	s := NewMemStorage()

	writeUserId, _ := s.GetUserId("marcel+client@email.com")
	readUserId, _ := s.GetUserId("marcel+client2@email.com")

	tests := []struct {
		name   string
		userId string
		op     Operation
		want   bool
	}{
		{"write user can run", writeUserId, Run, true},
		{"write user can stop", writeUserId, Stop, true},
		{"write user can get status", writeUserId, Status, true},
		{"write user can get output", writeUserId, Output, true},
		{"read user cannot run", readUserId, Run, false},
		{"read user cannot stop", readUserId, Stop, false},
		{"read user can get status", readUserId, Status, true},
		{"read user can get output", readUserId, Output, true},
		{"unknown user is denied", "nonexistent-id", Run, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.Authorized(tt.userId, tt.op)
			if got != tt.want {
				t.Errorf("Authorized(%q, %v) = %v, want %v", tt.userId, tt.op, got, tt.want)
			}
		})
	}
}

func TestSaveAndGetJob(t *testing.T) {
	s := NewMemStorage()
	job := NewJob()
	jobId := job.Id.String()

	s.SaveJob(jobId, job)

	got, ok := s.GetJob(jobId)
	if !ok {
		t.Fatal("GetJob returned not found for saved job")
	}
	if got.Id != job.Id {
		t.Errorf("GetJob returned wrong job: got %s, want %s", got.Id, job.Id)
	}

	_, ok = s.GetJob("nonexistent-id")
	if ok {
		t.Error("GetJob returned found for nonexistent job")
	}
}

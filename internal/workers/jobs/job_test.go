package jobs_test

import (
	"encoding/json"
	"testing"

	"github.com/kleffio/gameserver-daemon/internal/workers/jobs"
)

type testPayload struct {
	ServerName string `json:"server_name"`
	Version    string `json:"version"`
}

func TestNew(t *testing.T) {
	payload := testPayload{ServerName: "my-server", Version: "1.21.4"}

	job, err := jobs.New(jobs.JobTypeServerProvision, "resource-123", payload, 3)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if job.JobID == "" {
		t.Error("expected job_id to be set")
	}
	if job.JobType != jobs.JobTypeServerProvision {
		t.Errorf("expected job_type %s, got %s", jobs.JobTypeServerProvision, job.JobType)
	}
	if job.ResourceID != "resource-123" {
		t.Errorf("expected resource_id resource-123, got %s", job.ResourceID)
	}
	if job.Status != jobs.JobStatusPending {
		t.Errorf("expected status pending, got %s", job.Status)
	}
	if job.Attempts != 0 {
		t.Errorf("expected attempts 0, got %d", job.Attempts)
	}
	if job.MaxAttempts != 3 {
		t.Errorf("expected max_attempts 3, got %d", job.MaxAttempts)
	}
	if job.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
}

func TestUnmarshalPayload(t *testing.T) {
	payload := testPayload{ServerName: "my-server", Version: "1.21.4"}

	job, _ := jobs.New(jobs.JobTypeServerProvision, "resource-123", payload, 3)

	var result testPayload
	if err := job.UnmarshalPayload(&result); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ServerName != "my-server" {
		t.Errorf("expected server_name my-server, got %s", result.ServerName)
	}
}

func TestCanRetry(t *testing.T) {
	payload := testPayload{}
	job, _ := jobs.New(jobs.JobTypeServerProvision, "resource-123", payload, 3)

	job.Attempts = 2
	if !job.CanRetry() {
		t.Error("expected CanRetry true when attempts < max_attempts")
	}

	job.Attempts = 3
	if job.CanRetry() {
		t.Error("expected CanRetry false when attempts == max_attempts")
	}
}

func TestSerialization(t *testing.T) {
	payload := testPayload{ServerName: "my-server", Version: "1.21.4"}
	job, _ := jobs.New(jobs.JobTypeServerProvision, "resource-123", payload, 3)

	data, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("expected no error marshalling, got %v", err)
	}

	var result jobs.Job
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("expected no error unmarshalling, got %v", err)
	}
	if result.JobID != job.JobID {
		t.Errorf("expected job_id %s, got %s", job.JobID, result.JobID)
	}
}

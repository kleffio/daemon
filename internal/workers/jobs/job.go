package jobs

import (
	"encoding/json"
	"time"
)

type JobType string
type JobStatus string

const (
	JobTypeServerProvision JobType = "server.provision"
	JobTypeServerStart     JobType = "server.start"
	JobTypeServerStop      JobType = "server.stop"
	JobTypeServerDelete    JobType = "server.delete"
	JobTypeServerRestart   JobType = "server.restart"
)

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

type Job struct {
	JobID       string          `json:"job_id"`
	JobType     JobType         `json:"job_type"`
	ResourceID  string          `json:"resource_id"`
	Payload     json.RawMessage `json:"payload"`
	Status      JobStatus       `json:"status"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	CreatedAt   time.Time       `json:"created_at"`
}

func New(jobType JobType, resourceID string, payload any, maxAttempts int) (*Job, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &Job{
		JobID:       newID(),
		JobType:     jobType,
		ResourceID:  resourceID,
		Payload:     raw,
		Status:      JobStatusPending,
		Attempts:    0,
		MaxAttempts: maxAttempts,
		CreatedAt:   time.Now().UTC(),
	}, nil
}

func (j *Job) UnmarshalPayload(v any) error {
	return json.Unmarshal(j.Payload, v)
}

func (j *Job) CanRetry() bool {
	return j.Attempts < j.MaxAttempts
}

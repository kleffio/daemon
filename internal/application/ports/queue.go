package ports

import "github.com/kleffio/kleff-daemon/internal/workers/jobs"

type Queue interface {
	Enqueue(job *jobs.Job) error
	Dequeue() (*jobs.Job, error)
	Acknowledge(jobID string) error
	Retry(jobID string) error
	MarkFailed(jobID string) error
}

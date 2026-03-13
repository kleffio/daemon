package queue

import (
	"errors"
	"sync"

	"github.com/kleffio/gameserver-daemon/internal/workers/jobs"
)

var (
	ErrQueueEmpty  = errors.New("queue is empty")
	ErrJobNotFound = errors.New("job not found")
)

type MemoryQueue struct {
	mu         sync.Mutex
	pending    []*jobs.Job
	processing map[string]*jobs.Job
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		processing: make(map[string]*jobs.Job),
	}
}

func (q *MemoryQueue) Enqueue(job *jobs.Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job.Status = jobs.JobStatusPending
	q.pending = append(q.pending, job)
	return nil
}

func (q *MemoryQueue) Dequeue() (*jobs.Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.pending) == 0 {
		return nil, ErrQueueEmpty
	}

	job := q.pending[0]
	q.pending = q.pending[1:]

	job.Status = jobs.JobStatusProcessing
	job.Attempts++
	q.processing[job.JobID] = job

	return job, nil
}

func (q *MemoryQueue) Acknowledge(jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, ok := q.processing[jobID]
	if !ok {
		return ErrJobNotFound
	}

	job.Status = jobs.JobStatusCompleted
	delete(q.processing, jobID)
	return nil
}

func (q *MemoryQueue) Retry(jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, ok := q.processing[jobID]
	if !ok {
		return ErrJobNotFound
	}

	delete(q.processing, jobID)
	job.Status = jobs.JobStatusPending
	q.pending = append(q.pending, job)
	return nil
}

func (q *MemoryQueue) MarkFailed(jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, ok := q.processing[jobID]
	if !ok {
		return ErrJobNotFound
	}

	job.Status = jobs.JobStatusFailed
	delete(q.processing, jobID)
	return nil
}

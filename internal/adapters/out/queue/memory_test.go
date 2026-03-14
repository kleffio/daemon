package queue_test

import (
	"testing"

	"github.com/kleffio/gameserver-daemon/internal/adapters/out/queue"
	"github.com/kleffio/gameserver-daemon/internal/workers/jobs"
)

func newTestJob(t *testing.T) *jobs.Job {
	t.Helper()
	job, err := jobs.New(jobs.JobTypeServerProvision, "resource-123", map[string]string{}, 3)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}
	return job
}

func TestEnqueue(t *testing.T) {
	q := queue.NewMemoryQueue()
	job := newTestJob(t)

	if err := q.Enqueue(job); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDequeue(t *testing.T) {
	q := queue.NewMemoryQueue()
	job := newTestJob(t)
	q.Enqueue(job)

	result, err := q.Dequeue()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.JobID != job.JobID {
		t.Errorf("expected job %s, got %s", job.JobID, result.JobID)
	}
	if result.Status != jobs.JobStatusProcessing {
		t.Errorf("expected status processing, got %s", result.Status)
	}
	if result.Attempts != 1 {
		t.Errorf("expected attempts 1, got %d", result.Attempts)
	}
}

func TestDequeueEmpty(t *testing.T) {
	q := queue.NewMemoryQueue()

	_, err := q.Dequeue()
	if err != queue.ErrQueueEmpty {
		t.Errorf("expected ErrQueueEmpty, got %v", err)
	}
}

func TestAcknowledge(t *testing.T) {
	q := queue.NewMemoryQueue()
	job := newTestJob(t)
	q.Enqueue(job)
	q.Dequeue()

	if err := q.Acknowledge(job.JobID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if job.Status != jobs.JobStatusCompleted {
		t.Errorf("expected status completed, got %s", job.Status)
	}
}

func TestRetry(t *testing.T) {
	q := queue.NewMemoryQueue()
	job := newTestJob(t)
	q.Enqueue(job)
	q.Dequeue()

	if err := q.Retry(job.JobID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	result, err := q.Dequeue()
	if err != nil {
		t.Fatalf("expected job to be re-queued, got %v", err)
	}
	if result.JobID != job.JobID {
		t.Errorf("expected same job back, got %s", result.JobID)
	}
}

func TestMarkFailed(t *testing.T) {
	q := queue.NewMemoryQueue()
	job := newTestJob(t)
	q.Enqueue(job)
	q.Dequeue()

	if err := q.MarkFailed(job.JobID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if job.Status != jobs.JobStatusFailed {
		t.Errorf("expected status failed, got %s", job.Status)
	}
}

func TestJobLocking(t *testing.T) {
	q := queue.NewMemoryQueue()
	job := newTestJob(t)
	q.Enqueue(job)
	q.Dequeue()

	// job is processing, should not be dequeueable again
	_, err := q.Dequeue()
	if err != queue.ErrQueueEmpty {
		t.Errorf("expected queue to be empty while job is processing, got %v", err)
	}
}

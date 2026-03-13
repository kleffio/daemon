package workers_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kleffio/gameserver-daemon/internal/adapters/in/queue"
	"github.com/kleffio/gameserver-daemon/internal/workers"
	"github.com/kleffio/gameserver-daemon/internal/workers/jobs"
)

func TestDispatcherRegistersAndDispatchesJob(t *testing.T) {
	q := queue.NewMemoryQueue()
	d := workers.NewDispatcher(q, 2)

	var called atomic.Bool
	d.Register(jobs.JobTypeServerProvision, func(ctx context.Context, job *jobs.Job) error {
		called.Store(true)
		return nil
	})

	job, _ := jobs.New(jobs.JobTypeServerProvision, "resource-123", map[string]string{}, 3)
	q.Enqueue(job)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go d.Run(ctx)

	time.Sleep(200 * time.Millisecond)
	if !called.Load() {
		t.Error("expected handler to be called")
	}
}

func TestDispatcherMultipleWorkerTypes(t *testing.T) {
	q := queue.NewMemoryQueue()
	d := workers.NewDispatcher(q, 2)

	var provisionCalled, startCalled atomic.Bool

	d.Register(jobs.JobTypeServerProvision, func(ctx context.Context, job *jobs.Job) error {
		provisionCalled.Store(true)
		return nil
	})
	d.Register(jobs.JobTypeServerStart, func(ctx context.Context, job *jobs.Job) error {
		startCalled.Store(true)
		return nil
	})

	job1, _ := jobs.New(jobs.JobTypeServerProvision, "resource-1", map[string]string{}, 3)
	job2, _ := jobs.New(jobs.JobTypeServerStart, "resource-2", map[string]string{}, 3)
	q.Enqueue(job1)
	q.Enqueue(job2)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go d.Run(ctx)

	time.Sleep(200 * time.Millisecond)
	if !provisionCalled.Load() {
		t.Error("expected provision handler to be called")
	}
	if !startCalled.Load() {
		t.Error("expected start handler to be called")
	}
}

func TestDispatcherRetriesOnFailure(t *testing.T) {
	q := queue.NewMemoryQueue()
	d := workers.NewDispatcher(q, 1)

	var attempts atomic.Int32
	d.Register(jobs.JobTypeServerProvision, func(ctx context.Context, job *jobs.Job) error {
		attempts.Add(1)
		return errors.New("temporary failure")
	})

	job, _ := jobs.New(jobs.JobTypeServerProvision, "resource-123", map[string]string{}, 3)
	q.Enqueue(job)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go d.Run(ctx)

	time.Sleep(2 * time.Second)
	if attempts.Load() < 2 {
		t.Errorf("expected at least 2 attempts, got %d", attempts.Load())
	}
}

func TestDispatcherWorkerPoolConfigurable(t *testing.T) {
	q := queue.NewMemoryQueue()
	concurrency := 3
	d := workers.NewDispatcher(q, concurrency)

	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32

	d.Register(jobs.JobTypeServerProvision, func(ctx context.Context, job *jobs.Job) error {
		current := concurrent.Add(1)
		if current > maxConcurrent.Load() {
			maxConcurrent.Store(current)
		}
		time.Sleep(100 * time.Millisecond)
		concurrent.Add(-1)
		return nil
	})

	for i := 0; i < 6; i++ {
		job, _ := jobs.New(jobs.JobTypeServerProvision, "resource", map[string]string{}, 3)
		q.Enqueue(job)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go d.Run(ctx)

	time.Sleep(500 * time.Millisecond)
	if maxConcurrent.Load() > int32(concurrency) {
		t.Errorf("expected max concurrency %d, got %d", concurrency, maxConcurrent.Load())
	}
}

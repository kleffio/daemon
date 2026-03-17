package queue

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/kleffio/gameserver-daemon/internal/workers/jobs"
)

func setupRedis(t *testing.T) (*miniredis.Miniredis, *RedisQueue) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	queue, err := NewRedisQueue("redis://"+mr.Addr(), "", false)
	if err != nil {
		t.Fatalf("failed to init redis queue: %v", err)
	}
	return mr, queue
}

func TestRedisQueue_EnqueueDequeueAck(t *testing.T) {
	mr, q := setupRedis(t)
	defer mr.Close()

	job, _ := jobs.New(jobs.JobTypeServerStart, "res1", nil, 3)
	if err := q.Enqueue(job); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	dequeued, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue failed: %v", err)
	}

	if dequeued.JobID != job.JobID {
		t.Errorf("got job %s, want %s", dequeued.JobID, job.JobID)
	}

	if !mr.Exists(keyProcessing) {
		t.Errorf("job not in processing queue")
	}

	err = q.Acknowledge(dequeued.JobID)
	if err != nil {
		t.Fatalf("ack failed: %v", err)
	}

	if mr.Exists(keyProcessing) {
		t.Errorf("processing queue not clean")
	}
}

func TestRedisQueue_RetryToDelayed(t *testing.T) {
	mr, q := setupRedis(t)
	defer mr.Close()

	job, _ := jobs.New(jobs.JobTypeServerStart, "res1", nil, 5)
	q.Enqueue(job)

	dequeued, _ := q.Dequeue()
	
	if err := q.Retry(dequeued.JobID); err != nil {
		t.Fatalf("retry failed: %v", err)
	}

	if mr.Exists(keyProcessing) {
		t.Errorf("processing should be empty")
	}

	if !mr.Exists(keyDelayed) {
		t.Errorf("delayed queue should exist")
	}
	
	zset, _ := q.client.ZRange(context.Background(), keyDelayed, 0, -1).Result()
	if len(zset) != 1 {
		t.Errorf("expected 1 item in delayed, got %d", len(zset))
	}
}

func TestRedisQueue_MaxRetriesToDLQ(t *testing.T) {
	mr, q := setupRedis(t)
	defer mr.Close()

	job, _ := jobs.New(jobs.JobTypeServerStart, "res1", nil, 2) // Max attempts = 2
	q.Enqueue(job)

	d1, _ := q.Dequeue()
	

	d1.Attempts = 2
	
	q.Acknowledge(d1.JobID)
	q.Enqueue(d1)
	
	d2, _ := q.Dequeue()

	err := q.Retry(d2.JobID)
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}

	if mr.Exists(keyDelayed) {
		t.Errorf("should not be delayed")
	}

	if !mr.Exists(keyDead) {
		t.Errorf("should be in dead letter queue")
	}
}

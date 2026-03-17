package queue

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kleffio/gameserver-daemon/internal/workers/jobs"
	"github.com/redis/go-redis/v9"
)



const (
	keyPending    = "repo:queue:pending"
	keyProcessing = "repo:queue:processing"
	keyDelayed    = "repo:queue:delayed"
	keyDead       = "repo:queue:dead"
)


const luaPromoteDelayed = `
local jobs = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1], 'LIMIT', 0, 100)
for _, job in ipairs(jobs) do
    redis.call('ZREM', KEYS[1], job)
    redis.call('LPUSH', KEYS[2], job)
end
return #jobs
`

type RedisQueue struct {
	client *redis.Client
}

func NewRedisQueue(url, password string, useTLS bool) (*RedisQueue, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	opts.Password = password
	if useTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisQueue{
		client: client,
	}, nil
}

func (q *RedisQueue) Enqueue(job *jobs.Job) error {
	ctx := context.Background()

	job.Status = jobs.JobStatusPending

	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("serialize job: %w", err)
	}

	if err := q.client.LPush(ctx, keyPending, data).Err(); err != nil {
		return fmt.Errorf("redis lpush: %w", err)
	}
	return nil
}

func (q *RedisQueue) promoteDelayed(ctx context.Context) error {
	now := time.Now().UTC().Unix()
	return q.client.Eval(ctx, luaPromoteDelayed, []string{keyDelayed, keyPending}, now).Err()
}

func (q *RedisQueue) Dequeue() (*jobs.Job, error) {
	ctx := context.Background()

	if err := q.promoteDelayed(ctx); err != nil {
		return nil, fmt.Errorf("failed to promote delayed jobs: %w", err)
	}

	res, err := q.client.BLMove(ctx, keyPending, keyProcessing, "RIGHT", "LEFT", 2*time.Second).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrQueueEmpty
		}
		return nil, fmt.Errorf("failed to dequeue: %w", err)
	}

	var job jobs.Job
	if err := json.Unmarshal([]byte(res), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dequeued job: %w", err)
	}

	job.Status = jobs.JobStatusProcessing
	job.Attempts++

	return &job, nil
}


func (q *RedisQueue) removeByJobID(ctx context.Context, listKey, jobID string) (string, error) {
	items, err := q.client.LRange(ctx, listKey, 0, -1).Result()
	if err != nil {
		return "", err
	}
	for _, raw := range items {
		var j jobs.Job
		if err := json.Unmarshal([]byte(raw), &j); err == nil {
			if j.JobID == jobID {
				q.client.LRem(ctx, listKey, 1, raw)
				return raw, nil
			}
		}
	}
	return "", ErrJobNotFound
}

func (q *RedisQueue) Acknowledge(jobID string) error {
	ctx := context.Background()
	_, err := q.removeByJobID(ctx, keyProcessing, jobID)
	return err
}

func (q *RedisQueue) Retry(jobID string) error {
	ctx := context.Background()
	raw, err := q.removeByJobID(ctx, keyProcessing, jobID)
	if err != nil {
		return err
	}

	var job jobs.Job
	json.Unmarshal([]byte(raw), &job)

	if job.Attempts >= job.MaxAttempts {
		job.Status = jobs.JobStatusFailed
		newRaw, _ := json.Marshal(job)
		return q.client.LPush(ctx, keyDead, newRaw).Err()
	}

	job.Status = jobs.JobStatusPending
	delaySeconds := int64(job.Attempts * 5)
	runAt := time.Now().UTC().Unix() + delaySeconds

	newRaw, _ := json.Marshal(job)
	return q.client.ZAdd(ctx, keyDelayed, redis.Z{
		Score:  float64(runAt),
		Member: string(newRaw),
	}).Err()
}

func (q *RedisQueue) MarkFailed(jobID string) error {
	ctx := context.Background()
	raw, err := q.removeByJobID(ctx, keyProcessing, jobID)
	if err != nil {
		return err
	}

	var job jobs.Job
	json.Unmarshal([]byte(raw), &job)

	job.Status = jobs.JobStatusFailed
	newRaw, _ := json.Marshal(job)
	return q.client.LPush(ctx, keyDead, newRaw).Err()
}

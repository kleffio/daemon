package workers

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/kleffio/kleff-daemon/internal/application/ports"
	"github.com/kleffio/kleff-daemon/internal/workers/jobs"
)

type HandlerFunc func(ctx context.Context, job *jobs.Job) error

type Dispatcher struct {
	queue        ports.Queue
	handlers     map[jobs.JobType]HandlerFunc
	concurrency  int
	pollInterval time.Duration
}

func NewDispatcher(queue ports.Queue, concurrency int) *Dispatcher {
	return &Dispatcher{
		queue:        queue,
		handlers:     make(map[jobs.JobType]HandlerFunc),
		concurrency:  concurrency,
		pollInterval: 500 * time.Millisecond,
	}
}

func (d *Dispatcher) Register(jobType jobs.JobType, handler HandlerFunc) {
	d.handlers[jobType] = handler
}

func (d *Dispatcher) Run(ctx context.Context) {
	sem := make(chan struct{}, d.concurrency)
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		default:
		}

		job, err := d.queue.Dequeue()
		if err != nil {
			time.Sleep(d.pollInterval)
			continue
		}

		handler, ok := d.handlers[job.JobType]
		if !ok {
			d.queue.MarkFailed(job.JobID)
			continue
		}

		sem <- struct{}{}
		wg.Add(1)
		go func(j *jobs.Job, h HandlerFunc) {
			defer wg.Done()
			defer func() { <-sem }()

			err := h(ctx, j)
			if err != nil {
				if !errors.Is(err, ports.ErrPermanent) && j.CanRetry() {
					d.queue.Retry(j.JobID)
				} else {
					d.queue.MarkFailed(j.JobID)
				}
				return
			}

			d.queue.Acknowledge(j.JobID)
		}(job, handler)
	}
}

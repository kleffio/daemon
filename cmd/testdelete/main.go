package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kleffio/gameserver-daemon/internal/adapters/out/observability/logging"
	"github.com/kleffio/gameserver-daemon/internal/adapters/out/queue"
	"github.com/kleffio/gameserver-daemon/internal/adapters/out/repository/memory"
	k8sruntime "github.com/kleffio/gameserver-daemon/internal/adapters/out/runtime/kubernetes"
	"github.com/kleffio/gameserver-daemon/internal/workers"
	"github.com/kleffio/gameserver-daemon/internal/workers/jobs"
	"github.com/kleffio/gameserver-daemon/internal/workers/payloads"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	logger := logging.NewSlogAdapter()

	runtime, err := k8sruntime.New("http://127.0.0.1:8888", "default", "kleff-control-plane")
	if err != nil {
		log.Fatalf("failed to init kubernetes runtime: %v", err)
	}

	repo := memory.NewServerRepository()
	q := queue.NewMemoryQueue()

	deleteWorker := workers.NewDeleteWorker(runtime, repo, logger)
	dispatcher := workers.NewDispatcher(q, 1)
	dispatcher.Register(jobs.JobTypeServerDelete, deleteWorker.Handle)

	payload := payloads.ServerOperationPayload{
		OwnerID:  "owner-1",
		ServerID: "test-provision-2",
	}

	job, err := jobs.New(jobs.JobTypeServerDelete, "test-provision-2", payload, 3)
	if err != nil {
		log.Fatalf("failed to create job: %v", err)
	}

	q.Enqueue(job)

	fmt.Println("Delete job enqueued...")
	go dispatcher.Run(ctx)

	time.Sleep(30 * time.Second)
	fmt.Println("Done — check kubectl get gameserver and kubectl get pvc")
}

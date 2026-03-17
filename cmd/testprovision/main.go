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
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	logger := logging.NewSlogAdapter()

	runtime, err := k8sruntime.New("http://127.0.0.1:8888", "default", "kleff-control-plane")
	if err != nil {
		log.Fatalf("failed to init kubernetes runtime: %v", err)
	}

	repo := memory.NewServerRepository()
	q := queue.NewMemoryQueue()

	provisionWorker := workers.NewProvisionWorker(runtime, repo, logger)
	dispatcher := workers.NewDispatcher(q, 1)
	dispatcher.Register(jobs.JobTypeServerProvision, provisionWorker.Handle)

	payload := workers.ProvisionPayload{
		ServerName:   "test-provision-2",
		Type:         "PAPER",
		Version:      "1.21.4",
		MaxPlayers:   20,
		Difficulty:   "normal",
		Gamemode:     "survival",
		ViewDistance: 10,
		OnlineMode:   true,
		Memory:       "4Gi",
		Storage:      "10Gi",
	}

	job, err := jobs.New(jobs.JobTypeServerProvision, "test-provision-2", payload, 3)
	if err != nil {
		log.Fatalf("failed to create job: %v", err)
	}

	if err := q.Enqueue(job); err != nil {
		log.Fatalf("failed to enqueue job: %v", err)
	}

	fmt.Println("Job enqueued, waiting for server to reach Ready state...")
	go dispatcher.Run(ctx)

	time.Sleep(3 * time.Minute)

	record, err := repo.FindByID(ctx, payload.ServerName)
	if err != nil {
		log.Fatalf("failed to find server record: %v", err)
	}

	fmt.Printf("\nServer provisioned!\n")
	fmt.Printf("  ID:          %s\n", record.ID)
	fmt.Printf("  Name:        %s\n", record.Name)
	fmt.Printf("  Status:      %s\n", record.Status)
	fmt.Printf("  Node:        %s\n", record.NodeID)
	fmt.Printf("  RuntimeRef:  %s\n", record.RuntimeRef)
}

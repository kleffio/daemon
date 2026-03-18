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

	payload := payloads.ServerOperationPayload{
		OwnerID:     "owner-1",
		CrateID:     "test-provision-2",
		BlueprintID: "blueprint-1",
		Image:       "itzg/minecraft-server:latest",
		EnvOverrides: map[string]string{
			"TYPE":          "PAPER",
			"VERSION":       "1.21.4",
			"MAX_PLAYERS":   "20",
			"DIFFICULTY":    "normal",
			"MODE":          "survival",
			"VIEW_DISTANCE": "10",
			"LEVEL_SEED":    "",
			"ONLINE_MODE":   "true",
		},
		MemoryBytes: 4294967296,
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

	record, err := repo.FindByID(ctx, payload.CrateID)
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

package workers_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kleffio/kleff-daemon/internal/adapters/out/observability/logging"
	"github.com/kleffio/kleff-daemon/internal/application/ports"
	"github.com/kleffio/kleff-daemon/internal/workers"
	"github.com/kleffio/kleff-daemon/internal/workers/jobs"
	"github.com/kleffio/kleff-daemon/internal/workers/payloads"
	"github.com/kleffio/kleff-daemon/pkg/labels"
)

func TestProvisionWorkerHandleSuccess(t *testing.T) {
	runtime := &mockRuntime{
		returnServer: &ports.RunningServer{
			Labels: labels.ServerLabels{
				ServerID: "test-server",
				NodeID:   "test-node",
			},
			RuntimeRef: "test-server",
			State:      "Ready",
		},
	}
	repo := &mockRepository{}
	logger := logging.NewNoopLogger()

	worker := workers.NewProvisionWorker(runtime, repo, logger)

	payload := payloads.ServerOperationPayload{
		OwnerID:     "owner-1",
		ServerID:    "test-server",
		BlueprintID: "blueprint-1",
		Image:       "itzg/minecraft-server:latest",
		EnvOverrides: map[string]string{
			"TYPE":    "PAPER",
			"VERSION": "1.21.4",
		},
	}

	job, _ := jobs.New(jobs.JobTypeServerProvision, "test-server", payload, 3)

	if err := worker.Handle(context.Background(), job); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !runtime.startCalled {
		t.Error("expected runtime.Start to be called")
	}
	if !repo.saveCalled {
		t.Error("expected repository.Save to be called")
	}
	if repo.savedRecord.RuntimeRef != "test-server" {
		t.Errorf("expected runtime_ref test-server, got %s", repo.savedRecord.RuntimeRef)
	}
}

func TestProvisionWorkerHandleRuntimeFailure(t *testing.T) {
	runtime := &mockRuntime{
		returnErr: fmt.Errorf("agones unavailable"),
	}
	repo := &mockRepository{}
	logger := logging.NewNoopLogger()

	worker := workers.NewProvisionWorker(runtime, repo, logger)

	payload := payloads.ServerOperationPayload{
		OwnerID:     "owner-1",
		ServerID:    "test-server",
		BlueprintID: "blueprint-1",
		Image:       "itzg/minecraft-server:latest",
	}

	job, _ := jobs.New(jobs.JobTypeServerProvision, "test-server", payload, 3)

	if err := worker.Handle(context.Background(), job); err == nil {
		t.Error("expected error when runtime fails")
	}

	if repo.saveCalled {
		t.Error("expected repository.Save not to be called when runtime fails")
	}
}

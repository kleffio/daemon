package workers_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kleffio/kleff-daemon/internal/adapters/out/observability/logging"
	"github.com/kleffio/kleff-daemon/internal/application/ports"
	"github.com/kleffio/kleff-daemon/internal/workers"
	"github.com/kleffio/kleff-daemon/internal/workers/jobs"
	"github.com/kleffio/kleff-daemon/pkg/labels"
)

func TestStartWorkerHandleSuccess(t *testing.T) {
	runtime := &mockRuntime{
		returnServer: &ports.RunningServer{
			Labels: labels.WorkloadLabels{
				ServerID: "test-server",
				NodeID:   "test-node",
			},
			RuntimeRef: "test-server",
			State:      "Ready",
		},
	}
	repo := &mockRepository{}
	logger := logging.NewNoopLogger()

	worker := workers.NewStartWorker(runtime, repo, logger)

	spec := ports.WorkloadSpec{
		OwnerID:     "owner-1",
		ServerID:    "test-server",
		BlueprintID: "blueprint-1",
		ProjectID:   "proj-1",
		Image:       "itzg/minecraft-server:latest",
		EnvOverrides: map[string]string{
			"TYPE":    "PAPER",
			"VERSION": "1.21.4",
		},
	}

	job, _ := jobs.New(jobs.JobTypeServerStart, "test-server", spec, 3)

	if err := worker.Handle(context.Background(), job); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !runtime.startCalled {
		t.Error("expected runtime.Start to be called")
	}
}

func TestStartWorkerHandleRuntimeFailure(t *testing.T) {
	runtime := &mockRuntime{
		returnErr: fmt.Errorf("agones unavailable"),
	}
	repo := &mockRepository{}
	logger := logging.NewNoopLogger()

	worker := workers.NewStartWorker(runtime, repo, logger)

	spec := ports.WorkloadSpec{
		OwnerID:     "owner-1",
		ServerID:    "test-server",
		BlueprintID: "blueprint-1",
		ProjectID:   "proj-1",
		Image:       "itzg/minecraft-server:latest",
	}

	job, _ := jobs.New(jobs.JobTypeServerStart, "test-server", spec, 3)

	if err := worker.Handle(context.Background(), job); err == nil {
		t.Error("expected error when runtime fails")
	}
}

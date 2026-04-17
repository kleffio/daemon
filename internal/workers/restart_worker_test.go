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

func TestRestartWorkerHandleSuccess(t *testing.T) {
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

	worker := workers.NewRestartWorker(runtime, repo, logger, nil)

	spec := ports.WorkloadSpec{
		OwnerID:   "owner-1",
		ServerID:  "test-server",
		ProjectID: "proj-1",
	}

	job, _ := jobs.New(jobs.JobTypeServerRestart, "test-server", spec, 3)

	if err := worker.Handle(context.Background(), job); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !runtime.startCalled {
		t.Error("expected runtime.Start to be called")
	}
}

func TestRestartWorkerStopFailure(t *testing.T) {
	runtime := &mockRuntime{
		stopErr: fmt.Errorf("runtime unavailable"),
	}
	repo := &mockRepository{}
	logger := logging.NewNoopLogger()

	worker := workers.NewRestartWorker(runtime, repo, logger, nil)

	spec := ports.WorkloadSpec{
		OwnerID:   "owner-1",
		ServerID:  "test-server",
		ProjectID: "proj-1",
	}

	job, _ := jobs.New(jobs.JobTypeServerRestart, "test-server", spec, 3)

	if err := worker.Handle(context.Background(), job); err == nil {
		t.Error("expected error when stop fails")
	}
}

func TestRestartWorkerStartFailure(t *testing.T) {
	runtime := &mockRuntime{
		returnErr: fmt.Errorf("runtime unavailable"),
	}
	repo := &mockRepository{}
	logger := logging.NewNoopLogger()

	worker := workers.NewRestartWorker(runtime, repo, logger, nil)

	spec := ports.WorkloadSpec{
		OwnerID:   "owner-1",
		ServerID:  "test-server",
		ProjectID: "proj-1",
	}

	job, _ := jobs.New(jobs.JobTypeServerRestart, "test-server", spec, 3)

	if err := worker.Handle(context.Background(), job); err == nil {
		t.Error("expected error when start fails")
	}
}

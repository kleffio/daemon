package workers_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kleffio/kleff-daemon/internal/adapters/out/observability/logging"
	"github.com/kleffio/kleff-daemon/internal/application/ports"
	"github.com/kleffio/kleff-daemon/internal/workers"
	"github.com/kleffio/kleff-daemon/internal/workers/jobs"
)

func TestDeleteWorkerHandleSuccess(t *testing.T) {
	runtime := &mockRuntime{}
	repo := &mockRepository{}
	logger := logging.NewNoopLogger()

	worker := workers.NewDeleteWorker(runtime, repo, logger)

	spec := ports.WorkloadSpec{
		OwnerID:  "owner-1",
		ServerID: "test-server",
	}

	job, _ := jobs.New(jobs.JobTypeServerDelete, "test-server", spec, 3)

	if err := worker.Handle(context.Background(), job); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDeleteWorkerHandleRuntimeFailure(t *testing.T) {
	runtime := &mockRuntime{
		removeErr: fmt.Errorf("runtime unavailable"),
	}
	repo := &mockRepository{}
	logger := logging.NewNoopLogger()

	worker := workers.NewDeleteWorker(runtime, repo, logger)

	spec := ports.WorkloadSpec{
		OwnerID:  "owner-1",
		ServerID: "test-server",
	}

	job, _ := jobs.New(jobs.JobTypeServerDelete, "test-server", spec, 3)

	if err := worker.Handle(context.Background(), job); err == nil {
		t.Error("expected error when runtime fails")
	}
}

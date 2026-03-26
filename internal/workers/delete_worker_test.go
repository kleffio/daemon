package workers_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kleffio/kleff-daemon/internal/adapters/out/observability/logging"
	"github.com/kleffio/kleff-daemon/internal/workers"
	"github.com/kleffio/kleff-daemon/internal/workers/jobs"
	"github.com/kleffio/kleff-daemon/internal/workers/payloads"
)

func TestDeleteWorkerHandleSuccess(t *testing.T) {
	runtime := &mockRuntime{}
	repo := &mockRepository{}
	logger := logging.NewNoopLogger()

	worker := workers.NewDeleteWorker(runtime, repo, logger)

	payload := payloads.ServerOperationPayload{
		OwnerID:  "owner-1",
		ServerID: "test-server",
	}

	job, _ := jobs.New(jobs.JobTypeServerDelete, "test-server", payload, 3)

	if err := worker.Handle(context.Background(), job); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDeleteWorkerHandleRuntimeFailure(t *testing.T) {
	runtime := &mockRuntime{
		deleteErr: fmt.Errorf("agones unavailable"),
	}
	repo := &mockRepository{}
	logger := logging.NewNoopLogger()

	worker := workers.NewDeleteWorker(runtime, repo, logger)

	payload := payloads.ServerOperationPayload{
		OwnerID:  "owner-1",
		ServerID: "test-server",
	}

	job, _ := jobs.New(jobs.JobTypeServerDelete, "test-server", payload, 3)

	if err := worker.Handle(context.Background(), job); err == nil {
		t.Error("expected error when runtime fails")
	}
}

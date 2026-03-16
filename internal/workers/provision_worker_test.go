package workers_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kleffio/gameserver-daemon/internal/adapters/out/observability/logging"
	"github.com/kleffio/gameserver-daemon/internal/application/ports"
	"github.com/kleffio/gameserver-daemon/internal/workers"
	"github.com/kleffio/gameserver-daemon/internal/workers/jobs"
)

// Mock runtime
type mockRuntime struct {
	provisionCalled bool
	returnRecord    *ports.ServerRecord
	returnErr       error
}

func (m *mockRuntime) Provision(ctx context.Context, name string, p ports.ProvisionPayload) (*ports.ServerRecord, error) {
	m.provisionCalled = true
	return m.returnRecord, m.returnErr
}

// Mock repository
type mockRepository struct {
	saveCalled  bool
	savedRecord *ports.ServerRecord
	returnErr   error
}

func (m *mockRepository) Save(ctx context.Context, s *ports.ServerRecord) error {
	m.saveCalled = true
	m.savedRecord = s
	return m.returnErr
}

func (m *mockRepository) FindByID(ctx context.Context, id string) (*ports.ServerRecord, error) {
	return nil, nil
}

func (m *mockRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	return nil
}

func TestProvisionWorkerHandleSuccess(t *testing.T) {
	runtime := &mockRuntime{
		returnRecord: &ports.ServerRecord{
			ID:         "test-id",
			Name:       "test-server",
			Status:     "provisioning",
			Runtime:    "agones",
			RuntimeRef: "test-server",
		},
	}
	repo := &mockRepository{}
	logger := logging.NewNoopLogger()

	worker := workers.NewProvisionWorker(runtime, repo, logger)

	payload := workers.ProvisionPayload{
		ServerName:   "test-server",
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

	job, _ := jobs.New(jobs.JobTypeServerProvision, "test-server", payload, 3)

	if err := worker.Handle(context.Background(), job); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !runtime.provisionCalled {
		t.Error("expected runtime.Provision to be called")
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

	payload := workers.ProvisionPayload{
		ServerName: "test-server",
		Type:       "PAPER",
		Version:    "1.21.4",
		Memory:     "4Gi",
		Storage:    "10Gi",
	}

	job, _ := jobs.New(jobs.JobTypeServerProvision, "test-server", payload, 3)

	if err := worker.Handle(context.Background(), job); err == nil {
		t.Error("expected error when runtime fails")
	}

	if repo.saveCalled {
		t.Error("expected repository.Save not to be called when runtime fails")
	}
}

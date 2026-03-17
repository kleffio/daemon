package workers_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kleffio/gameserver-daemon/internal/adapters/out/observability/logging"
	"github.com/kleffio/gameserver-daemon/internal/application/ports"
	"github.com/kleffio/gameserver-daemon/internal/workers"
	"github.com/kleffio/gameserver-daemon/internal/workers/jobs"
	"github.com/kleffio/gameserver-daemon/pkg/labels"
)

type mockRuntime struct {
	startCalled bool
	returnCrate *ports.RunningCrate
	returnErr   error
}

func (m *mockRuntime) Start(ctx context.Context, name string, p ports.ProvisionPayload) (*ports.RunningCrate, error) {
	m.startCalled = true
	return m.returnCrate, m.returnErr
}

func (m *mockRuntime) Stop(ctx context.Context, crateID string) error   { return nil }
func (m *mockRuntime) Delete(ctx context.Context, crateID string) error { return nil }
func (m *mockRuntime) GetByID(ctx context.Context, crateID string) (*ports.RunningCrate, error) {
	return nil, nil
}
func (m *mockRuntime) Reconcile(ctx context.Context, nodeID string) ([]*ports.RunningCrate, error) {
	return nil, nil
}
func (m *mockRuntime) Stats(ctx context.Context, crateID string) (*ports.RawStats, error) {
	return nil, nil
}

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
		returnCrate: &ports.RunningCrate{
			Labels: labels.CrateLabels{
				CrateID: "test-server",
				NodeID:  "test-node",
			},
			RuntimeRef: "test-server",
			State:      "Ready",
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

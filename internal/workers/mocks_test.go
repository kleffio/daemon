package workers_test

import (
	"context"

	"github.com/kleffio/gameserver-daemon/internal/application/ports"
	"github.com/kleffio/gameserver-daemon/internal/workers/payloads"
)

type mockRuntime struct {
	startCalled bool
	returnCrate *ports.RunningCrate
	returnErr   error
	deleteErr   error
}

func (m *mockRuntime) Start(ctx context.Context, payload payloads.ServerOperationPayload) (*ports.RunningCrate, error) {
	m.startCalled = true
	return m.returnCrate, m.returnErr
}

func (m *mockRuntime) Stop(ctx context.Context, crateID string) error { return nil }
func (m *mockRuntime) Delete(ctx context.Context, crateID string) error {
	return m.deleteErr
}
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

package workers_test

import (
	"context"
	"io"

	"github.com/kleffio/kleff-daemon/internal/application/ports"
)

type mockRuntime struct {
	deployCalled bool
	startCalled  bool
	returnServer *ports.RunningServer
	returnErr    error
	removeErr    error
	stopErr      error
}

func (m *mockRuntime) Deploy(ctx context.Context, spec ports.WorkloadSpec) (*ports.RunningServer, error) {
	m.deployCalled = true
	return m.returnServer, m.returnErr
}

func (m *mockRuntime) Start(ctx context.Context, spec ports.WorkloadSpec) (*ports.RunningServer, error) {
	m.startCalled = true
	return m.returnServer, m.returnErr
}

func (m *mockRuntime) Stop(ctx context.Context, workloadID string) error  { return m.stopErr }
func (m *mockRuntime) Remove(ctx context.Context, workloadID string) error { return m.removeErr }
func (m *mockRuntime) Status(ctx context.Context, workloadID string) (*ports.WorkloadHealth, error) {
	return nil, nil
}
func (m *mockRuntime) Endpoint(ctx context.Context, workloadID string) (string, error) {
	return "", nil
}
func (m *mockRuntime) Logs(ctx context.Context, workloadID string, follow bool) (io.ReadCloser, error) {
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

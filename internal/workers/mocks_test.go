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

func (m *mockRuntime) EnsureProjectScope(_ context.Context, projectID, projectSlug string) (*ports.ProjectScope, error) {
	return &ports.ProjectScope{ProjectID: projectID, ProjectSlug: projectSlug}, nil
}
func (m *mockRuntime) TeardownProjectScope(_ context.Context, _ string) error { return nil }
func (m *mockRuntime) Stop(_ context.Context, _, _ string) error               { return m.stopErr }
func (m *mockRuntime) Remove(_ context.Context, _, _ string) error             { return m.removeErr }
func (m *mockRuntime) Status(_ context.Context, _, workloadID string) (*ports.WorkloadHealth, error) {
	return nil, nil
}
func (m *mockRuntime) Endpoint(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (m *mockRuntime) Logs(_ context.Context, _, _ string, _ bool) (io.ReadCloser, error) {
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

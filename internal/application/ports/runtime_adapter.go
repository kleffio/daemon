package ports

import (
	"context"
	"io"
)

// RuntimeAdapter replaces ContainerRuntime as the generic interface for deploying
// workloads on either Docker or Kubernetes.
type RuntimeAdapter interface {
	Deploy(ctx context.Context, spec WorkloadSpec) (*RunningServer, error)
	Remove(ctx context.Context, workloadID string) error
	Start(ctx context.Context, workloadID string) error
	Stop(ctx context.Context, workloadID string) error
	Status(ctx context.Context, workloadID string) (*WorkloadHealth, error)
	Endpoint(ctx context.Context, workloadID string) (string, error)
	Logs(ctx context.Context, workloadID string, follow bool) (io.ReadCloser, error)
}

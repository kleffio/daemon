package ports

import (
	"context"
	"io"
)

// RuntimeAdapter is the generic interface for deploying workloads on any backend.
type RuntimeAdapter interface {
	// Deploy provisions and starts a new workload from scratch.
	Deploy(ctx context.Context, spec WorkloadSpec) (*RunningServer, error)
	// Start resumes a previously stopped workload. The full spec is required
	// because some backends (e.g. Agones) are stateless and need it to recreate resources.
	Start(ctx context.Context, spec WorkloadSpec) (*RunningServer, error)
	// Stop suspends a running workload without removing it.
	Stop(ctx context.Context, workloadID string) error
	// Remove permanently deletes a workload and all its resources.
	Remove(ctx context.Context, workloadID string) error
	// Status returns the current health and resource usage of a workload.
	Status(ctx context.Context, workloadID string) (*WorkloadHealth, error)
	// Endpoint returns the host:port address players/users connect to.
	// primaryPort is the container-side port number to look up (e.g. 25565 for Minecraft).
	Endpoint(ctx context.Context, workloadID string, primaryPort int) (string, error)
	// Logs streams the workload's stdout/stderr.
	Logs(ctx context.Context, workloadID string, follow bool) (io.ReadCloser, error)
}

package ports

import (
	"context"

	"github.com/kleffio/kleff-daemon/pkg/labels"
)

type RunningServer struct {
	Labels     labels.WorkloadLabels
	RuntimeRef string
	State      string
}

type RawStats struct {
	CPUMillicores float64
	MemoryBytes   int64
	NetBytesIn    int64
	NetBytesOut   int64
	ActivePlayers int
}

type ContainerRuntime interface {
	Start(ctx context.Context, payload WorkloadSpec) (*RunningServer, error)
	Stop(ctx context.Context, serverID string) error
	Delete(ctx context.Context, serverID string) error
	GetByID(ctx context.Context, serverID string) (*RunningServer, error)
	Reconcile(ctx context.Context, nodeID string) ([]*RunningServer, error)
	Stats(ctx context.Context, serverID string) (*RawStats, error)
}

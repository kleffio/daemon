package ports

import "context"

// WorkloadRepository persists workload state on the local node.
type WorkloadRepository interface {
	Save(ctx context.Context, health *WorkloadHealth) error
	FindByID(ctx context.Context, workloadID string) (*WorkloadHealth, error)
	UpdateState(ctx context.Context, workloadID string, state string) error
	Delete(ctx context.Context, workloadID string) error
	ListByNode(ctx context.Context, nodeID string) ([]*WorkloadHealth, error)
}

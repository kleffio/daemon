package ports

import (
	"context"
	"time"
)

// NodeRecord holds the locally-persisted metadata for this agent node.
type NodeRecord struct {
	NodeID   string
	Region   string
	LastSeen time.Time
}

// NodeRepository persists node-level metadata.
type NodeRepository interface {
	Save(ctx context.Context, node *NodeRecord) error
	FindByID(ctx context.Context, nodeID string) (*NodeRecord, error)
}

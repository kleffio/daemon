package ports

import (
	"context"

	"github.com/kleffio/gameserver-daemon/pkg/labels"
)

type ProvisionPayload struct {
	ServerName   string
	Type         string
	Version      string
	MaxPlayers   int
	Difficulty   string
	Gamemode     string
	ViewDistance int
	WorldSeed    string
	OnlineMode   bool
	Memory       string
	Storage      string
}

type RunningCrate struct {
	Labels     labels.CrateLabels
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
	Start(ctx context.Context, name string, payload ProvisionPayload) (*RunningCrate, error)
	Stop(ctx context.Context, crateID string) error
	Delete(ctx context.Context, crateID string) error
	GetByID(ctx context.Context, crateID string) (*RunningCrate, error)
	Reconcile(ctx context.Context, nodeID string) ([]*RunningCrate, error)
	Stats(ctx context.Context, crateID string) (*RawStats, error)
}

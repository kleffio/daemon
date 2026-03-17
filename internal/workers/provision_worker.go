package workers

import (
	"context"
	"fmt"

	"github.com/kleffio/gameserver-daemon/internal/application/ports"
	"github.com/kleffio/gameserver-daemon/internal/workers/jobs"
)

type ProvisionPayload struct {
	ServerName   string `json:"server_name"`
	Type         string `json:"type"`
	Version      string `json:"version"`
	MaxPlayers   int    `json:"max_players"`
	Difficulty   string `json:"difficulty"`
	Gamemode     string `json:"gamemode"`
	ViewDistance int    `json:"view_distance"`
	WorldSeed    string `json:"world_seed"`
	OnlineMode   bool   `json:"online_mode"`
	Memory       string `json:"memory"`
	Storage      string `json:"storage"`
}

type ProvisionWorker struct {
	runtime    ports.ContainerRuntime
	repository ports.ServerRepository
	logger     ports.Logger
}

func NewProvisionWorker(runtime ports.ContainerRuntime, repository ports.ServerRepository, logger ports.Logger) *ProvisionWorker {
	return &ProvisionWorker{
		runtime:    runtime,
		repository: repository,
		logger:     logger,
	}
}

func (w *ProvisionWorker) Handle(ctx context.Context, job *jobs.Job) error {
	log := w.logger.With(
		ports.LogKeyJobID, job.JobID,
		ports.LogKeyWorkerType, "server.provision",
	)

	var payload ProvisionPayload
	if err := job.UnmarshalPayload(&payload); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	log.Info("Provisioning server", ports.LogKeyServerID, payload.ServerName)

	crate, err := w.runtime.Start(ctx, payload.ServerName, ports.ProvisionPayload{
		ServerName:   payload.ServerName,
		Type:         payload.Type,
		Version:      payload.Version,
		MaxPlayers:   payload.MaxPlayers,
		Difficulty:   payload.Difficulty,
		Gamemode:     payload.Gamemode,
		ViewDistance: payload.ViewDistance,
		WorldSeed:    payload.WorldSeed,
		OnlineMode:   payload.OnlineMode,
		Memory:       payload.Memory,
		Storage:      payload.Storage,
	})
	if err != nil {
		log.Error("Failed to provision server", err)
		return fmt.Errorf("provision failed: %w", err)
	}

	record := &ports.ServerRecord{
		ID:         payload.ServerName,
		Name:       payload.ServerName,
		Status:     crate.State,
		NodeID:     crate.Labels.NodeID,
		Runtime:    "agones",
		RuntimeRef: crate.RuntimeRef,
	}

	if err := w.repository.Save(ctx, record); err != nil {
		log.Error("Failed to store server record", err)
		return fmt.Errorf("failed to store runtime reference: %w", err)
	}

	log.Info("Server provisioned successfully", ports.LogKeyServerID, record.ID, "runtime_ref", record.RuntimeRef)
	return nil
}

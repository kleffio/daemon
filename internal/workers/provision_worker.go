package workers

import (
	"context"
	"fmt"

	"github.com/kleffio/gameserver-daemon/internal/application/ports"
	"github.com/kleffio/gameserver-daemon/internal/workers/jobs"
	"github.com/kleffio/gameserver-daemon/internal/workers/payloads"
)

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

	var payload payloads.ServerOperationPayload
	if err := job.UnmarshalPayload(&payload); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	log.Info("Provisioning server", ports.LogKeyServerID, payload.ServerID)

	server, err := w.runtime.Provision(ctx, payload)
	if err != nil {
		log.Error("Failed to provision server", err)
		return fmt.Errorf("provision failed: %w", err)
	}

	record := &ports.ServerRecord{
		ID:         payload.ServerID,
		Name:       payload.ServerID,
		Status:     server.State,
		NodeID:     server.Labels.NodeID,
		Runtime:    "agones",
		RuntimeRef: server.RuntimeRef,
	}

	if err := w.repository.Save(ctx, record); err != nil {
		log.Error("Failed to store server record", err)
		return fmt.Errorf("failed to store runtime reference: %w", err)
	}

	log.Info("Server provisioned successfully", ports.LogKeyServerID, record.ID, "runtime_ref", record.RuntimeRef)
	return nil
}

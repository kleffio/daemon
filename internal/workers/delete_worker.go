package workers

import (
	"context"
	"fmt"

	"github.com/kleffio/gameserver-daemon/internal/application/ports"
	"github.com/kleffio/gameserver-daemon/internal/workers/jobs"
	"github.com/kleffio/gameserver-daemon/internal/workers/payloads"
)

type DeleteWorker struct {
	runtime    ports.ContainerRuntime
	repository ports.ServerRepository
	logger     ports.Logger
}

func NewDeleteWorker(runtime ports.ContainerRuntime, repository ports.ServerRepository, logger ports.Logger) *DeleteWorker {
	return &DeleteWorker{
		runtime:    runtime,
		repository: repository,
		logger:     logger,
	}
}

func (w *DeleteWorker) Handle(ctx context.Context, job *jobs.Job) error {
	log := w.logger.With(
		ports.LogKeyJobID, job.JobID,
		ports.LogKeyWorkerType, "server.delete",
	)

	var payload payloads.ServerOperationPayload
	if err := job.UnmarshalPayload(&payload); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	log.Info("Deleting server", ports.LogKeyServerID, payload.CrateID)

	if err := w.runtime.Delete(ctx, payload.CrateID); err != nil {
		log.Error("Failed to delete server", err)
		return fmt.Errorf("delete failed: %w", err)
	}

	if err := w.repository.UpdateStatus(ctx, payload.CrateID, "deleted"); err != nil {
		log.Warn("Failed to update server status after delete", "crate_id", payload.CrateID)
	}

	log.Info("Server deleted successfully", ports.LogKeyServerID, payload.CrateID)
	return nil
}

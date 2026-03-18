package workers

import (
	"context"
	"fmt"

	"github.com/kleffio/gameserver-daemon/internal/application/ports"
	"github.com/kleffio/gameserver-daemon/internal/workers/jobs"
	"github.com/kleffio/gameserver-daemon/internal/workers/payloads"
)

type StopWorker struct {
	runtime    ports.ContainerRuntime
	repository ports.ServerRepository
	logger     ports.Logger
}

func NewStopWorker(runtime ports.ContainerRuntime, repository ports.ServerRepository, logger ports.Logger) *StopWorker {
	return &StopWorker{
		runtime:    runtime,
		repository: repository,
		logger:     logger,
	}
}

func (w *StopWorker) Handle(ctx context.Context, job *jobs.Job) error {
	log := w.logger.With(
		ports.LogKeyJobID, job.JobID,
		ports.LogKeyWorkerType, "server.stop",
	)

	var payload payloads.ServerOperationPayload
	if err := job.UnmarshalPayload(&payload); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	log.Info("Stopping server", ports.LogKeyServerID, payload.CrateID)

	if err := w.runtime.Stop(ctx, payload.CrateID); err != nil {
		log.Error("Failed to stop server", err)
		return fmt.Errorf("stop failed: %w", err)
	}

	if err := w.repository.UpdateStatus(ctx, payload.CrateID, "stopped"); err != nil {
		log.Warn("Failed to update server status after stop", "crate_id", payload.CrateID)
	}

	log.Info("Server stopped successfully", ports.LogKeyServerID, payload.CrateID)
	return nil
}

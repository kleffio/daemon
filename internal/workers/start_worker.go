package workers

import (
	"context"
	"fmt"

	"github.com/kleffio/gameserver-daemon/internal/application/ports"
	"github.com/kleffio/gameserver-daemon/internal/workers/jobs"
	"github.com/kleffio/gameserver-daemon/internal/workers/payloads"
)

type StartWorker struct {
	runtime    ports.ContainerRuntime
	repository ports.ServerRepository
	logger     ports.Logger
}

func NewStartWorker(runtime ports.ContainerRuntime, repository ports.ServerRepository, logger ports.Logger) *StartWorker {
	return &StartWorker{
		runtime:    runtime,
		repository: repository,
		logger:     logger,
	}
}

func (w *StartWorker) Handle(ctx context.Context, job *jobs.Job) error {
	log := w.logger.With(
		ports.LogKeyJobID, job.JobID,
		ports.LogKeyWorkerType, "server.start",
	)

	var payload payloads.ServerOperationPayload
	if err := job.UnmarshalPayload(&payload); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	log.Info("Starting server", ports.LogKeyServerID, payload.CrateID)

	crate, err := w.runtime.Start(ctx, payload)
	if err != nil {
		log.Error("Failed to start server", err)
		return fmt.Errorf("start failed: %w", err)
	}

	if err := w.repository.UpdateStatus(ctx, payload.CrateID, crate.State); err != nil {
		log.Warn("Failed to update server status after start", "crate_id", payload.CrateID)
	}

	log.Info("Server started successfully", ports.LogKeyServerID, payload.CrateID)
	return nil
}

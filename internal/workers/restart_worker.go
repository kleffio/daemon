package workers

import (
	"context"
	"fmt"

	"github.com/kleffio/kleff-daemon/internal/application/ports"
	"github.com/kleffio/kleff-daemon/internal/workers/jobs"
	"github.com/kleffio/kleff-daemon/internal/workers/payloads"
)

type RestartWorker struct {
	runtime    ports.ContainerRuntime
	repository ports.ServerRepository
	logger     ports.Logger
}

func NewRestartWorker(runtime ports.ContainerRuntime, repository ports.ServerRepository, logger ports.Logger) *RestartWorker {
	return &RestartWorker{
		runtime:    runtime,
		repository: repository,
		logger:     logger,
	}
}

func (w *RestartWorker) Handle(ctx context.Context, job *jobs.Job) error {
	log := w.logger.With(
		ports.LogKeyJobID, job.JobID,
		ports.LogKeyWorkerType, "server.restart",
	)

	var payload payloads.ServerOperationPayload
	if err := job.UnmarshalPayload(&payload); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	log.Info("Restarting server", ports.LogKeyServerID, payload.ServerID)

	if err := w.runtime.Stop(ctx, payload.ServerID); err != nil {
		log.Error("Failed to stop server during restart", err)
		return fmt.Errorf("restart failed on stop: %w", err)
	}

	server, err := w.runtime.Start(ctx, payload)
	if err != nil {
		log.Error("Failed to start server during restart", err)
		return fmt.Errorf("restart failed on start: %w", err)
	}

	if err := w.repository.UpdateStatus(ctx, payload.ServerID, server.State); err != nil {
		log.Warn("Failed to update server status after restart", "server_id", payload.ServerID)
	}

	log.Info("Server restarted successfully", ports.LogKeyServerID, payload.ServerID)
	return nil
}

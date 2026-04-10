package workers

import (
	"context"
	"fmt"

	"github.com/kleffio/kleff-daemon/internal/application/ports"
	"github.com/kleffio/kleff-daemon/internal/workers/jobs"
)

type RestartWorker struct {
	runtime    ports.RuntimeAdapter
	repository ports.ServerRepository
	logger     ports.Logger
}

func NewRestartWorker(runtime ports.RuntimeAdapter, repository ports.ServerRepository, logger ports.Logger) *RestartWorker {
	return &RestartWorker{runtime: runtime, repository: repository, logger: logger}
}

func (w *RestartWorker) Handle(ctx context.Context, job *jobs.Job) error {
	log := w.logger.With(ports.LogKeyJobID, job.JobID, ports.LogKeyWorkerType, "server.restart")

	var spec ports.WorkloadSpec
	if err := job.UnmarshalPayload(&spec); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	log.Info("Restarting server", ports.LogKeyServerID, spec.ServerID)

	if err := w.runtime.Stop(ctx, spec.ServerID); err != nil {
		log.Error("Failed to stop server during restart", err)
		return fmt.Errorf("restart failed on stop: %w", err)
	}

	server, err := w.runtime.Start(ctx, spec)
	if err != nil {
		log.Error("Failed to start server during restart", err)
		return fmt.Errorf("restart failed on start: %w", err)
	}

	if err := w.repository.UpdateStatus(ctx, spec.ServerID, server.State); err != nil {
		log.Warn("Failed to update server status after restart", "server_id", spec.ServerID)
	}

	log.Info("Server restarted successfully", ports.LogKeyServerID, spec.ServerID)
	return nil
}

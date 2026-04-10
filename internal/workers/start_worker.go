package workers

import (
	"context"
	"fmt"

	"github.com/kleffio/kleff-daemon/internal/application/ports"
	"github.com/kleffio/kleff-daemon/internal/workers/jobs"
)

type StartWorker struct {
	runtime    ports.RuntimeAdapter
	repository ports.ServerRepository
	logger     ports.Logger
}

func NewStartWorker(runtime ports.RuntimeAdapter, repository ports.ServerRepository, logger ports.Logger) *StartWorker {
	return &StartWorker{runtime: runtime, repository: repository, logger: logger}
}

func (w *StartWorker) Handle(ctx context.Context, job *jobs.Job) error {
	log := w.logger.With(ports.LogKeyJobID, job.JobID, ports.LogKeyWorkerType, "server.start")

	var spec ports.WorkloadSpec
	if err := job.UnmarshalPayload(&spec); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	log.Info("Starting server", ports.LogKeyServerID, spec.ServerID)

	server, err := w.runtime.Start(ctx, spec)
	if err != nil {
		log.Error("Failed to start server", err)
		return fmt.Errorf("start failed: %w", err)
	}

	if err := w.repository.UpdateStatus(ctx, spec.ServerID, server.State); err != nil {
		log.Warn("Failed to update server status after start", "server_id", spec.ServerID)
	}

	log.Info("Server started successfully", ports.LogKeyServerID, spec.ServerID)
	return nil
}

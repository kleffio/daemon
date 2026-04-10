package workers

import (
	"context"
	"fmt"

	"github.com/kleffio/kleff-daemon/internal/application/ports"
	"github.com/kleffio/kleff-daemon/internal/workers/jobs"
)

type StopWorker struct {
	runtime    ports.RuntimeAdapter
	repository ports.ServerRepository
	logger     ports.Logger
}

func NewStopWorker(runtime ports.RuntimeAdapter, repository ports.ServerRepository, logger ports.Logger) *StopWorker {
	return &StopWorker{runtime: runtime, repository: repository, logger: logger}
}

func (w *StopWorker) Handle(ctx context.Context, job *jobs.Job) error {
	log := w.logger.With(ports.LogKeyJobID, job.JobID, ports.LogKeyWorkerType, "server.stop")

	var spec ports.WorkloadSpec
	if err := job.UnmarshalPayload(&spec); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	log.Info("Stopping server", ports.LogKeyServerID, spec.ServerID)

	if err := w.runtime.Stop(ctx, spec.ServerID); err != nil {
		log.Error("Failed to stop server", err)
		return fmt.Errorf("stop failed: %w", err)
	}

	if err := w.repository.UpdateStatus(ctx, spec.ServerID, "stopped"); err != nil {
		log.Warn("Failed to update server status after stop", "server_id", spec.ServerID)
	}

	log.Info("Server stopped successfully", ports.LogKeyServerID, spec.ServerID)
	return nil
}

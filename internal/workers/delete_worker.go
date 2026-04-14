package workers

import (
	"context"
	"fmt"
	"strings"

	"github.com/kleffio/kleff-daemon/internal/application/ports"
	"github.com/kleffio/kleff-daemon/internal/workers/jobs"
)

type DeleteWorker struct {
	runtime    ports.RuntimeAdapter
	repository ports.ServerRepository
	logger     ports.Logger
}

func NewDeleteWorker(runtime ports.RuntimeAdapter, repository ports.ServerRepository, logger ports.Logger) *DeleteWorker {
	return &DeleteWorker{runtime: runtime, repository: repository, logger: logger}
}

func (w *DeleteWorker) Handle(ctx context.Context, job *jobs.Job) error {
	log := w.logger.With(ports.LogKeyJobID, job.JobID, ports.LogKeyWorkerType, "server.delete")

	var spec ports.WorkloadSpec
	if err := job.UnmarshalPayload(&spec); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	log.Info("Deleting server", ports.LogKeyServerID, spec.ServerID)

	if err := w.runtime.Remove(ctx, spec.ServerID); err != nil {
		if strings.Contains(err.Error(), "container not found") {
			log.Info("Container already gone, treating delete as complete", ports.LogKeyServerID, spec.ServerID)
		} else {
			log.Error("Failed to delete server", err)
			return fmt.Errorf("delete failed: %w", err)
		}
	}

	if err := w.repository.UpdateStatus(ctx, spec.ServerID, "deleted"); err != nil {
		log.Warn("Failed to update server status after delete", "server_id", spec.ServerID)
	}

	log.Info("Server deleted successfully", ports.LogKeyServerID, spec.ServerID)
	return nil
}

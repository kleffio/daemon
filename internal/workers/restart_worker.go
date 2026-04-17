package workers

import (
	"context"
	"fmt"
	platformclient "github.com/kleffio/kleff-daemon/internal/adapters/out/platform"
	"github.com/kleffio/kleff-daemon/internal/application/ports"
	"github.com/kleffio/kleff-daemon/internal/workers/jobs"
)

type RestartWorker struct {
	runtime        ports.RuntimeAdapter
	repository     ports.ServerRepository
	logger         ports.Logger
	platformClient *platformclient.Client
}

func NewRestartWorker(runtime ports.RuntimeAdapter, repository ports.ServerRepository, logger ports.Logger, platformClient *platformclient.Client) *RestartWorker {
	return &RestartWorker{runtime: runtime, repository: repository, logger: logger, platformClient: platformClient}
}

func (w *RestartWorker) Handle(ctx context.Context, job *jobs.Job) error {
	log := w.logger.With(ports.LogKeyJobID, job.JobID, ports.LogKeyWorkerType, "server.restart")

	var spec ports.WorkloadSpec
	if err := job.UnmarshalPayload(&spec); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	log.Info("Restarting server", ports.LogKeyServerID, spec.ServerID)

	// Tell the platform we're restarting so the UI can show it.
	if w.platformClient != nil {
		if err := w.platformClient.ReportStatus(ctx, spec.ServerID, "restarting"); err != nil {
			log.Error("Failed to report restarting status to platform", err)
		}
	}

	if err := w.runtime.Stop(ctx, spec.ProjectID, spec.ServerID); err != nil {
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

	// Docker assigns a new random port on each start — report the updated address.
	if w.platformClient != nil {
		if address, err := w.runtime.Endpoint(ctx, spec.ProjectID, spec.ServerID); err != nil {
			log.Error("Failed to get endpoint after restart", err)
			if err := w.platformClient.ReportStatus(ctx, spec.ServerID, "succeeded"); err != nil {
				log.Error("Failed to report status to platform", err)
			}
		} else {
			if err := w.platformClient.ReportAddress(ctx, spec.ServerID, address); err != nil {
				log.Error("Failed to report address to platform after restart — falling back to status-only update", err)
				if err := w.platformClient.ReportStatus(ctx, spec.ServerID, "succeeded"); err != nil {
					log.Error("Failed to report succeeded status to platform", err)
				}
			} else {
				log.Info("Address reported to platform after restart", ports.LogKeyServerID, spec.ServerID, "address", address)
			}
		}
	}

	log.Info("Server restarted successfully", ports.LogKeyServerID, spec.ServerID)
	return nil
}

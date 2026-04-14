package workers

import (
	"context"
	"fmt"
	platformclient "github.com/kleffio/kleff-daemon/internal/adapters/out/platform"
	"github.com/kleffio/kleff-daemon/internal/application/ports"
	"github.com/kleffio/kleff-daemon/internal/workers/jobs"
)

type StartWorker struct {
	runtime        ports.RuntimeAdapter
	repository     ports.ServerRepository
	logger         ports.Logger
	platformClient *platformclient.Client
}

func NewStartWorker(runtime ports.RuntimeAdapter, repository ports.ServerRepository, logger ports.Logger, platformClient *platformclient.Client) *StartWorker {
	return &StartWorker{runtime: runtime, repository: repository, logger: logger, platformClient: platformClient}
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

	primaryPort := 0
	if len(spec.PortRequirements) > 0 {
		primaryPort = spec.PortRequirements[0].TargetPort
	}
	if address, err := w.runtime.Endpoint(ctx, spec.ServerID, primaryPort); err != nil {
		log.Error("Failed to get endpoint after start — reporting succeeded without address", err)
		if err := w.platformClient.ReportStatus(ctx, spec.ServerID, "succeeded"); err != nil {
			log.Error("Failed to report status to platform", err)
		}
	} else {
		if err := w.platformClient.ReportAddress(ctx, spec.ServerID, address); err != nil {
			log.Error("Failed to report address to platform after start — falling back to status-only update", err)
			if err := w.platformClient.ReportStatus(ctx, spec.ServerID, "succeeded"); err != nil {
				log.Error("Failed to report succeeded status to platform", err)
			}
		} else {
			log.Info("Address reported to platform after start", ports.LogKeyServerID, spec.ServerID, "address", address)
		}
	}

	log.Info("Server started successfully", ports.LogKeyServerID, spec.ServerID)
	return nil
}

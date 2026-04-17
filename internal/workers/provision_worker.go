package workers

import (
	"context"
	"fmt"
	"strings"

	platformclient "github.com/kleffio/kleff-daemon/internal/adapters/out/platform"
	"github.com/kleffio/kleff-daemon/internal/application/ports"
	"github.com/kleffio/kleff-daemon/internal/workers/jobs"
)

type ProvisionWorker struct {
	runtime        ports.RuntimeAdapter
	repository     ports.ServerRepository
	logger         ports.Logger
	platformClient *platformclient.Client
}

func NewProvisionWorker(runtime ports.RuntimeAdapter, repository ports.ServerRepository, logger ports.Logger, platformClient *platformclient.Client) *ProvisionWorker {
	return &ProvisionWorker{runtime: runtime, repository: repository, logger: logger, platformClient: platformClient}
}

func (w *ProvisionWorker) Handle(ctx context.Context, job *jobs.Job) error {
	log := w.logger.With(ports.LogKeyJobID, job.JobID, ports.LogKeyWorkerType, "server.provision")

	var spec ports.WorkloadSpec
	if err := job.UnmarshalPayload(&spec); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	log.Info("Provisioning server", ports.LogKeyServerID, spec.ServerID)

	server, err := w.runtime.Deploy(ctx, spec)
	if err != nil {
		log.Error("Failed to provision server", err)
		if strings.Contains(err.Error(), "Invalid container name") {
			return fmt.Errorf("provision failed (bad name): %w: %w", err, ports.ErrPermanent)
		}
		return fmt.Errorf("provision failed: %w", err)
	}

	record := &ports.ServerRecord{
		ID:         spec.ServerID,
		Name:       spec.ServerID,
		Status:     server.State,
		NodeID:     server.Labels.NodeID,
		RuntimeRef: server.RuntimeRef,
	}

	if err := w.repository.Save(ctx, record); err != nil {
		log.Error("Failed to store server record", err)
		return fmt.Errorf("failed to store runtime reference: %w", err)
	}

	log.Info("Server provisioned successfully", ports.LogKeyServerID, record.ID, "runtime_ref", record.RuntimeRef)

	if w.platformClient == nil {
		return nil
	}

	// Always mark the deployment as succeeded — the server is running.
	// Attempt to get the address too; if that fails the status still updates.
	address, err := w.runtime.Endpoint(ctx, spec.ProjectID, spec.ServerID)
	if err != nil {
		log.Error("Failed to get server endpoint — reporting succeeded without address", err)
		if err := w.platformClient.ReportStatus(ctx, spec.ServerID, "succeeded"); err != nil {
			log.Error("Failed to report succeeded status to platform", err)
		}
		return nil
	}

	// ReportAddress updates both the address and status to succeeded in one call.
	if err := w.platformClient.ReportAddress(ctx, spec.ServerID, address); err != nil {
		log.Error("Failed to report address to platform — falling back to status-only update", err)
		if err := w.platformClient.ReportStatus(ctx, spec.ServerID, "succeeded"); err != nil {
			log.Error("Failed to report succeeded status to platform", err)
		}
	} else {
		log.Info("Address reported to platform", ports.LogKeyServerID, spec.ServerID, "address", address)
	}

	return nil
}

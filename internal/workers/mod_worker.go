package workers

import (
	"context"
	"fmt"

	"github.com/kleffio/kleff-daemon/internal/application/ports"
	"github.com/kleffio/kleff-daemon/internal/workers/jobs"
)

type ModWorker struct {
	runtime ports.RuntimeAdapter
	logger  ports.Logger
}

func NewModWorker(runtime ports.RuntimeAdapter, logger ports.Logger) *ModWorker {
	return &ModWorker{runtime: runtime, logger: logger}
}

func (w *ModWorker) HandleInstall(ctx context.Context, job *jobs.Job) error {
	log := w.logger.With(ports.LogKeyJobID, job.JobID, ports.LogKeyWorkerType, "server.install_mod")

	var spec ports.ModInstallSpec
	if err := job.UnmarshalPayload(&spec); err != nil {
		return fmt.Errorf("%w: invalid payload: %s", ports.ErrPermanent, err)
	}
	if spec.ServerID == "" || spec.ProjectID == "" {
		return fmt.Errorf("%w: server_id and project_id are required", ports.ErrPermanent)
	}
	if spec.DownloadURL == "" || spec.FileName == "" || spec.ContentType == "" {
		return fmt.Errorf("%w: download_url, file_name, and content_type are required", ports.ErrPermanent)
	}
	if spec.StoragePath == "" {
		spec.StoragePath = "/data"
	}

	log.Info("Installing mod", "server_id", spec.ServerID, "file", spec.FileName, "type", spec.ContentType)

	if err := w.runtime.InjectFile(ctx, spec.ProjectID, spec.ServerID, spec.StoragePath, spec.ContentType, spec.DownloadURL, spec.FileName); err != nil {
		log.Error("Failed to inject mod file", err)
		return fmt.Errorf("inject file: %w", err)
	}

	log.Info("Mod installed successfully", "server_id", spec.ServerID, "file", spec.FileName)
	return nil
}

func (w *ModWorker) HandleUninstall(ctx context.Context, job *jobs.Job) error {
	log := w.logger.With(ports.LogKeyJobID, job.JobID, ports.LogKeyWorkerType, "server.uninstall_mod")

	var spec ports.ModUninstallSpec
	if err := job.UnmarshalPayload(&spec); err != nil {
		return fmt.Errorf("%w: invalid payload: %s", ports.ErrPermanent, err)
	}
	if spec.ServerID == "" || spec.ProjectID == "" {
		return fmt.Errorf("%w: server_id and project_id are required", ports.ErrPermanent)
	}
	if spec.FileName == "" || spec.ContentType == "" {
		return fmt.Errorf("%w: file_name and content_type are required", ports.ErrPermanent)
	}
	if spec.StoragePath == "" {
		spec.StoragePath = "/data"
	}

	log.Info("Uninstalling mod", "server_id", spec.ServerID, "file", spec.FileName, "type", spec.ContentType)

	if err := w.runtime.RemoveFile(ctx, spec.ProjectID, spec.ServerID, spec.StoragePath, spec.ContentType, spec.FileName); err != nil {
		log.Error("Failed to remove mod file", err)
		return fmt.Errorf("remove file: %w", err)
	}

	log.Info("Mod uninstalled successfully", "server_id", spec.ServerID, "file", spec.FileName)
	return nil
}

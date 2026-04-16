package ports

import "context"

type WorkloadStatusUpdate struct {
	WorkloadID   string
	ProjectID    string
	Status       string
	RuntimeRef   string
	Endpoint     string
	NodeID       string
	ErrorMessage string
}

type WorkloadStatusReporter interface {
	ReportStatus(ctx context.Context, update WorkloadStatusUpdate) error
}

type NoopWorkloadStatusReporter struct{}

func (NoopWorkloadStatusReporter) ReportStatus(_ context.Context, _ WorkloadStatusUpdate) error {
	return nil
}

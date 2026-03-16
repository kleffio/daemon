package ports

import "context"

type ServerRepository interface {
	Save(ctx context.Context, server *ServerRecord) error
	FindByID(ctx context.Context, id string) (*ServerRecord, error)
	UpdateStatus(ctx context.Context, id string, status string) error
}

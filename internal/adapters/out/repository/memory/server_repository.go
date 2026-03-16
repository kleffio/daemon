package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kleffio/gameserver-daemon/internal/application/ports"
)

type ServerRepository struct {
	db *sql.DB
}

func NewServerRepository(db *sql.DB) *ServerRepository {
	return &ServerRepository{db: db}
}

func (r *ServerRepository) Save(ctx context.Context, s *ports.ServerRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO servers (id, name, address, port, status, node_id, runtime, runtime_ref, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Name, s.Address, s.Port, s.Status, s.NodeID, s.Runtime, s.RuntimeRef,
		time.Now().UTC(), time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to save server: %w", err)
	}
	return nil
}

func (r *ServerRepository) FindByID(ctx context.Context, id string) (*ports.ServerRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, address, port, status, node_id, runtime, runtime_ref
		FROM servers WHERE id = ?`, id)

	var s ports.ServerRecord
	if err := row.Scan(&s.ID, &s.Name, &s.Address, &s.Port, &s.Status, &s.NodeID, &s.Runtime, &s.RuntimeRef); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("server not found: %s", id)
		}
		return nil, fmt.Errorf("failed to find server: %w", err)
	}
	return &s, nil
}

func (r *ServerRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE servers SET status = ?, updated_at = ? WHERE id = ?`,
		status, time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("failed to update server status: %w", err)
	}
	return nil
}

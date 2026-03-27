package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/kleffio/kleff-daemon/internal/application/ports"
)

type ServerRepository struct {
	mu      sync.Mutex
	servers map[string]*ports.ServerRecord
}

func NewServerRepository() *ServerRepository {
	return &ServerRepository{
		servers: make(map[string]*ports.ServerRecord),
	}
}

func (r *ServerRepository) Save(ctx context.Context, s *ports.ServerRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.servers[s.ID] = s
	return nil
}

func (r *ServerRepository) FindByID(ctx context.Context, id string) (*ports.ServerRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.servers[id]
	if !ok {
		return nil, fmt.Errorf("server not found: %s", id)
	}
	return s, nil
}

func (r *ServerRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.servers[id]
	if !ok {
		return fmt.Errorf("server not found: %s", id)
	}
	s.Status = status
	return nil
}

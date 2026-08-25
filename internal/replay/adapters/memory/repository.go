package memory

import (
	"context"
	"sync"

	"proxynth/payment-sandbox/internal/replay/domain"
)

var _ interface {
	FindByID(context.Context, domain.ScenarioID) (*domain.Scenario, error)
} = (*Repository)(nil)

type Repository struct {
	mu        sync.RWMutex
	scenarios map[domain.ScenarioID]*domain.Scenario
}

func NewRepository() *Repository {
	return &Repository{scenarios: make(map[domain.ScenarioID]*domain.Scenario)}
}

func (r *Repository) FindByID(ctx context.Context, id domain.ScenarioID) (*domain.Scenario, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.scenarios[id], nil
}

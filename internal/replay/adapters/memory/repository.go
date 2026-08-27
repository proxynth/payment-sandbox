package memory

import (
	"context"
	"sync"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	"proxynth/payment-sandbox/internal/replay/domain"
)

var _ interface {
	Save(context.Context, *domain.Scenario) error
	FindByID(context.Context, domain.ScenarioID) (*domain.Scenario, error)
} = (*Repository)(nil)

type Repository struct {
	mu        sync.RWMutex
	scenarios map[domain.ScenarioID]*domain.Scenario
}

func (r *Repository) Save(ctx context.Context, scenario *domain.Scenario) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if scenario == nil {
		return domain.ErrInvalidScenarioID
	}
	if err := scenario.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.scenarios[scenario.ID]; exists {
		return domain.ErrScenarioAlreadyExists
	}
	copy := *scenario
	copy.InitialPayments = append([]paymentdomain.PaymentState(nil), scenario.InitialPayments...)
	copy.Commands = append([]domain.Command(nil), scenario.Commands...)
	r.scenarios[scenario.ID] = &copy
	return nil
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

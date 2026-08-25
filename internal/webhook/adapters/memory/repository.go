package memory

import (
	"context"
	"sync"

	"proxynth/payment-sandbox/internal/webhook/application"
	"proxynth/payment-sandbox/internal/webhook/domain"
)

var _ application.Repository = (*Repository)(nil)

type Repository struct {
	mu        sync.RWMutex
	endpoints map[domain.EndpointID]*domain.Endpoint
}

func NewRepository() *Repository {
	return &Repository{endpoints: make(map[domain.EndpointID]*domain.Endpoint)}
}

func (r *Repository) Save(ctx context.Context, endpoint *domain.Endpoint) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.endpoints[endpoint.ID()]; exists {
		return application.ErrEndpointAlreadyExists
	}
	r.endpoints[endpoint.ID()] = endpoint
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id domain.EndpointID) (*domain.Endpoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	endpoint, exists := r.endpoints[id]
	if !exists {
		return nil, application.ErrEndpointNotFound
	}
	return endpoint, nil
}

func (r *Repository) List(ctx context.Context) ([]*domain.Endpoint, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	endpoints := make([]*domain.Endpoint, 0, len(r.endpoints))
	for _, endpoint := range r.endpoints {
		endpoints = append(endpoints, endpoint)
	}
	return endpoints, nil
}

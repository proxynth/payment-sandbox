package application

import (
	"context"

	"proxynth/payment-sandbox/internal/webhook/domain"
)

type Repository interface {
	Save(context.Context, *domain.Endpoint) error
	FindByID(context.Context, domain.EndpointID) (*domain.Endpoint, error)
	List(context.Context) ([]*domain.Endpoint, error)
}

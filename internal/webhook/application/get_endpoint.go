package application

import (
	"context"

	"proxynth/payment-sandbox/internal/webhook/domain"
)

type GetEndpoint struct{ repository Repository }

func NewGetEndpoint(repository Repository) *GetEndpoint {
	return &GetEndpoint{repository: repository}
}

func (g *GetEndpoint) Execute(ctx context.Context, id domain.EndpointID) (*domain.Endpoint, error) {
	return g.repository.FindByID(ctx, id)
}

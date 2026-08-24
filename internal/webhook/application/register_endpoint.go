package application

import (
	"context"

	"proxynth/payment-sandbox/internal/webhook/domain"
)

type RegisterEndpointCommand struct {
	ID  domain.EndpointID
	URL string
}

type RegisterEndpoint struct{ repository Repository }

func NewRegisterEndpoint(repository Repository) *RegisterEndpoint {
	return &RegisterEndpoint{repository: repository}
}

func (r *RegisterEndpoint) Execute(ctx context.Context, command RegisterEndpointCommand) (*domain.Endpoint, error) {
	endpoint, err := domain.NewEndpoint(command.ID, command.URL)
	if err != nil {
		return nil, err
	}

	if err := r.repository.Save(ctx, endpoint); err != nil {
		return nil, err
	}

	return endpoint, nil
}

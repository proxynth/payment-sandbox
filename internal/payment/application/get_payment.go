package application

import (
	"context"

	"proxynth/payment-sandbox/internal/payment/domain"
)

type GetPayment struct {
	repository Repository
}

func NewGetPayment(repository Repository) *GetPayment {
	return &GetPayment{repository: repository}
}

func (g *GetPayment) Execute(ctx context.Context, id domain.ID) (*domain.Payment, error) {
	return g.repository.FindByID(ctx, id)
}

package application

import (
	"context"

	"proxynth/payment-sandbox/internal/payment/domain"
)

type Repository interface {
	Save(ctx context.Context, payment *domain.Payment) error
	FindByID(ctx context.Context, id domain.ID) (*domain.Payment, error)
}

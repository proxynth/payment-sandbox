package application

import (
	"context"

	"proxynth/payment-sandbox/internal/payment/domain"
)

type fakeRepository struct {
	payment *domain.Payment

	findErr error
	saveErr error

	saveCalls int
}

func (r *fakeRepository) FindByID(
	_ context.Context,
	_ domain.ID,
) (*domain.Payment, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}

	return r.payment, nil
}

func (r *fakeRepository) Save(
	_ context.Context,
	payment *domain.Payment,
) error {
	r.saveCalls++

	if r.saveErr != nil {
		return r.saveErr
	}

	r.payment = payment

	return nil
}

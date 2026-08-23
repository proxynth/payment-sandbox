package application

import (
	"context"

	"proxynth/payment-sandbox/internal/payment/domain"
)

type CancelPaymentCommand struct {
	PaymentID domain.ID
}

type CancelPayment struct {
	repository Repository
}

func NewCancelPayment(repository Repository) *CancelPayment {
	return &CancelPayment{
		repository: repository,
	}
}

func (c *CancelPayment) Execute(
	ctx context.Context,
	command CancelPaymentCommand,
) (*domain.Payment, error) {
	payment, err := c.repository.FindByID(ctx, command.PaymentID)
	if err != nil {
		return nil, err
	}

	if err := payment.Cancel(); err != nil {
		return nil, err
	}

	if err := c.repository.Save(ctx, payment); err != nil {
		return nil, err
	}

	return payment, nil
}

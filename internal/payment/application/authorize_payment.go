package application

import (
	"context"

	"proxynth/payment-sandbox/internal/payment/domain"
)

type AuthorizePaymentCommand struct {
	PaymentID domain.ID
}

type AuthorizePayment struct {
	repository Repository
}

func NewAuthorizePayment(repository Repository) *AuthorizePayment {
	return &AuthorizePayment{
		repository: repository,
	}
}

func (c *AuthorizePayment) Execute(
	ctx context.Context,
	command AuthorizePaymentCommand,
) (*domain.Payment, error) {
	payment, err := c.repository.FindByID(
		ctx,
		command.PaymentID,
	)
	if err != nil {
		return nil, err
	}

	if err := payment.Authorize(); err != nil {
		return nil, err
	}

	if err := c.repository.Save(ctx, payment); err != nil {
		return nil, err
	}

	return payment, nil
}

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
	publisher  EventPublisher
}

func NewCancelPayment(repository Repository) *CancelPayment {
	return &CancelPayment{
		repository: repository,
	}
}

func NewCancelPaymentWithPublisher(repository Repository, publisher EventPublisher) *CancelPayment {
	return &CancelPayment{repository: repository, publisher: publisher}
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
	if c.publisher != nil {
		if err := c.publisher.Publish(ctx, payment, domain.EventPaymentCancelled); err != nil {
			return nil, err
		}
	}

	return payment, nil
}

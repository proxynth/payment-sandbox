package application

import (
	"context"

	"proxynth/payment-sandbox/internal/payment/domain"
)

type FailPaymentCommand struct {
	PaymentID domain.ID
}

type FailPayment struct {
	repository Repository
	publisher  EventPublisher
}

func NewFailPayment(repository Repository) *FailPayment {
	return &FailPayment{
		repository: repository,
	}
}

func NewFailPaymentWithPublisher(repository Repository, publisher EventPublisher) *FailPayment {
	return &FailPayment{repository: repository, publisher: publisher}
}

func (c *FailPayment) Execute(
	ctx context.Context,
	command FailPaymentCommand,
) (*domain.Payment, error) {
	payment, err := c.repository.FindByID(ctx, command.PaymentID)
	if err != nil {
		return nil, err
	}

	if err := payment.Fail(); err != nil {
		return nil, err
	}

	if err := c.repository.Save(ctx, payment); err != nil {
		return nil, err
	}
	if c.publisher != nil {
		if err := c.publisher.Publish(ctx, payment, domain.EventPaymentFailed); err != nil {
			return nil, err
		}
	}

	return payment, nil
}

package application

import (
	"context"

	"proxynth/payment-sandbox/internal/payment/domain"
)

type CreatePaymentCommand struct {
	ID       domain.ID
	Amount   int64
	Currency domain.Currency
}

type CreatePayment struct {
	repository Repository
	publisher  EventPublisher
}

func NewCreatePayment(repository Repository) *CreatePayment {
	return &CreatePayment{
		repository: repository,
	}
}

func NewCreatePaymentWithPublisher(repository Repository, publisher EventPublisher) *CreatePayment {
	return &CreatePayment{repository: repository, publisher: publisher}
}

func (c *CreatePayment) Execute(
	ctx context.Context,
	command CreatePaymentCommand,
) (*domain.Payment, error) {
	amount, err := domain.NewMoney(
		command.Amount,
		command.Currency,
	)
	if err != nil {
		return nil, err
	}

	payment, err := domain.New(
		command.ID,
		amount,
	)
	if err != nil {
		return nil, err
	}

	if err := c.repository.Save(ctx, payment); err != nil {
		return nil, err
	}
	if c.publisher != nil {
		if err := c.publisher.Publish(ctx, payment, domain.EventPaymentCreated); err != nil {
			return nil, err
		}
	}

	return payment, nil
}

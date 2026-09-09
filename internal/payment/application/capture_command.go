package application

import (
	"context"

	"proxynth/payment-sandbox/internal/payment/domain"
)

type CapturePaymentCommand struct {
	PaymentID domain.ID
	Amount    int64
	Currency  domain.Currency
}

type CapturePayment struct {
	repository Repository
	publisher  EventPublisher
}

func NewCapturePayment(repository Repository) *CapturePayment {
	return &CapturePayment{
		repository: repository,
	}
}

func NewCapturePaymentWithPublisher(repository Repository, publisher EventPublisher) *CapturePayment {
	return &CapturePayment{repository: repository, publisher: publisher}
}

func (c *CapturePayment) Execute(
	ctx context.Context,
	command CapturePaymentCommand,
) (*domain.Payment, error) {
	var result *domain.Payment
	err := withinRepositoryTransaction(ctx, c.repository, func(txctx context.Context) error { var err error; result, err = c.execute(txctx, command); return err })
	return result, err
}

func (c *CapturePayment) execute(ctx context.Context, command CapturePaymentCommand) (*domain.Payment, error) {
	payment, err := c.repository.FindByID(ctx, command.PaymentID)
	if err != nil {
		return nil, err
	}

	amount, err := domain.NewMoney(command.Amount, command.Currency)
	if err != nil {
		return nil, err
	}

	if err := payment.Capture(amount); err != nil {
		return nil, err
	}

	if err := c.repository.Save(ctx, payment); err != nil {
		return nil, err
	}
	if c.publisher != nil {
		if err := c.publisher.Publish(ctx, payment, domain.EventPaymentCaptured); err != nil {
			return nil, err
		}
	}

	return payment, nil
}

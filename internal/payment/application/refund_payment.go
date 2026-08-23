package application

import (
	"context"

	"proxynth/payment-sandbox/internal/payment/domain"
)

type RefundPaymentCommand struct {
	PaymentID domain.ID
	Amount    int64
	Currency  domain.Currency
}

type RefundPayment struct {
	repository Repository
}

func NewRefundPayment(repository Repository) *RefundPayment {
	return &RefundPayment{
		repository: repository,
	}
}

func (c *RefundPayment) Execute(
	ctx context.Context,
	command RefundPaymentCommand,
) (*domain.Payment, error) {
	payment, err := c.repository.FindByID(ctx, command.PaymentID)
	if err != nil {
		return nil, err
	}

	amount, err := domain.NewMoney(
		command.Amount,
		command.Currency,
	)
	if err != nil {
		return nil, err
	}

	if err := payment.Refund(amount); err != nil {
		return nil, err
	}

	if err := c.repository.Save(ctx, payment); err != nil {
		return nil, err
	}

	return payment, nil
}

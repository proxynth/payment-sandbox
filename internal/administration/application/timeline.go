package application

import (
	"context"
	"errors"

	paymentapplication "proxynth/payment-sandbox/internal/payment/application"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
)

var (
	ErrNilPaymentRepository = errors.New("timeline payment repository is nil")
	ErrNilEventLog          = errors.New("timeline event log is nil")
)

type Timeline struct {
	payments paymentapplication.Repository
	events   paymentapplication.EventLog
}

type PaymentTimeline struct {
	Payment *paymentdomain.Payment
	Events  []paymentdomain.BusinessEvent
}

func NewTimeline(
	payments paymentapplication.Repository,
	events paymentapplication.EventLog,
) (*Timeline, error) {
	if payments == nil {
		return nil, ErrNilPaymentRepository
	}
	if events == nil {
		return nil, ErrNilEventLog
	}

	return &Timeline{payments: payments, events: events}, nil
}

func (t *Timeline) Execute(ctx context.Context, id paymentdomain.ID) (PaymentTimeline, error) {
	payment, err := t.payments.FindByID(ctx, id)
	if err != nil {
		return PaymentTimeline{}, err
	}

	events, err := t.events.ListByAggregate(ctx, id)
	if err != nil {
		return PaymentTimeline{}, err
	}

	return PaymentTimeline{Payment: payment, Events: events}, nil
}

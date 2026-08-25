package application

import (
	"context"
	"errors"
	"time"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	"proxynth/payment-sandbox/internal/saga/domain"
)

type Store interface {
	Save(context.Context, domain.Instance) error
	Find(context.Context, domain.ID) (domain.Instance, error)
}

type Publisher interface {
	Publish(context.Context, domain.Message) error
}

type Executor interface {
	Execute(context.Context, domain.Message) (Outcome, error)
}

type Outcome string

const (
	OutcomeSucceeded Outcome = "succeeded"
	OutcomePending   Outcome = "pending"
	OutcomeFailed    Outcome = "failed"
)

func (o Outcome) Valid() bool {
	return o == OutcomeSucceeded || o == OutcomePending || o == OutcomeFailed
}

type Orchestrator struct {
	store     Store
	publisher Publisher
	clock     func() time.Time
}

func NewOrchestrator(store Store, publisher Publisher, now func() time.Time) (*Orchestrator, error) {
	if store == nil || publisher == nil || now == nil {
		return nil, errors.New("invalid saga orchestrator")
	}
	return &Orchestrator{store: store, publisher: publisher, clock: now}, nil
}

func (o *Orchestrator) Start(ctx context.Context, id domain.ID, paymentID string, seed uint64) error {
	return o.StartWithPayload(ctx, id, paymentID, nil, seed)
}

func (o *Orchestrator) StartWithPayload(ctx context.Context, id domain.ID, paymentID string, payload []byte, seed uint64) error {
	at := o.clock().UTC()
	instance, err := domain.NewWithPayload(id, paymentdomain.ID(paymentID), payload, seed, at)
	if err != nil {
		return err
	}
	if err := o.store.Save(ctx, instance); err != nil {
		return err
	}
	return o.publishCurrent(ctx, instance, at)
}

// Handle is idempotent: a completed message is acknowledged without running
// it again, which makes duplicate at-least-once delivery safe.
func (o *Orchestrator) Handle(ctx context.Context, message domain.Message, executor Executor) error {
	if err := message.Validate(); err != nil {
		return err
	}
	instance, err := o.store.Find(ctx, message.SagaID)
	if err != nil {
		return err
	}
	if instance.Status == domain.StatusCompleted || instance.Status == domain.StatusFailed {
		return nil
	}
	if instance.CurrentStep != message.Step {
		return nil
	}
	outcome, err := executor.Execute(ctx, message)
	if err != nil {
		return err
	}
	if !outcome.Valid() {
		return errors.New("invalid saga execution outcome")
	}
	if outcome == OutcomePending {
		return nil
	}
	at := o.clock().UTC()
	if instance.Status == domain.StatusCompensating {
		if outcome != OutcomeSucceeded {
			return errors.New("compensation failed")
		}
		if err := instance.ApplyCompensationSuccess(message.Step, at); err != nil {
			return err
		}
	} else if outcome == OutcomeSucceeded {
		if err := instance.ApplySuccess(message.Step, at); err != nil {
			return err
		}
	} else {
		if err := instance.BeginCompensation(message.Step, at); err != nil {
			return err
		}
	}
	if err := o.store.Save(ctx, instance); err != nil {
		return err
	}
	if instance.Status == domain.StatusRunning || instance.Status == domain.StatusCompensating {
		return o.publishCurrent(ctx, instance, at)
	}
	return nil
}

func (o *Orchestrator) publishCurrent(ctx context.Context, instance domain.Instance, at time.Time) error {
	return o.publisher.Publish(ctx, domain.Message{
		ID:     string(instance.ID) + ":" + string(instance.CurrentStep) + ":" + itoa(instance.Version),
		SagaID: instance.ID, PaymentID: instance.PaymentID, Step: instance.CurrentStep,
		Payload: append([]byte(nil), instance.Payload...),
		Seed:    instance.Seed, ScheduledAt: at, VirtualAt: at, Attempt: 1,
	})
}

func itoa(value uint64) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var result [20]byte
	index := len(result)
	for value > 0 {
		index--
		result[index] = digits[value%10]
		value /= 10
	}
	return string(result[index:])
}

package application

import (
	"context"
	"fmt"
	"sort"
	"time"

	paymentapplication "proxynth/payment-sandbox/internal/payment/application"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	"proxynth/payment-sandbox/internal/platform/clock"
	replaydomain "proxynth/payment-sandbox/internal/replay/domain"
)

// Result contains the deterministic state produced by one scenario execution.
type Result struct {
	ScenarioID                 replaydomain.ScenarioID
	Provider                   replaydomain.ProviderConfiguration
	DeterministicConfiguration replaydomain.DeterministicConfiguration
	CurrentVirtualTime         time.Time
	Payments                   []paymentdomain.PaymentState
}

// Runner executes one scenario against the production payment application
// services and an execution-scoped in-memory repository.
type Runner struct{}

func NewRunner() *Runner {
	return &Runner{}
}

func (r *Runner) Run(ctx context.Context, scenario replaydomain.Scenario) (Result, error) {
	if err := scenario.Validate(); err != nil {
		return Result{}, err
	}

	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	virtualClock, err := clock.NewVirtualClock(scenario.InitialVirtualTime)
	if err != nil {
		return Result{}, err
	}

	repository := newMemoryRepository()
	for _, state := range scenario.InitialPayments {
		payment, err := paymentdomain.Restore(state)
		if err != nil {
			return Result{}, err
		}

		repository.payments[payment.ID()] = payment
	}

	services := commandServices{repository: repository}
	for index, command := range scenario.Commands {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}

		if err := services.execute(ctx, command); err != nil {
			return Result{}, fmt.Errorf(
				"execute scenario command %d (%s): %w",
				index,
				command.Type,
				err,
			)
		}
	}

	return Result{
		ScenarioID:                 scenario.ID,
		Provider:                   scenario.Provider,
		DeterministicConfiguration: scenario.DeterministicConfiguration,
		CurrentVirtualTime:         virtualClock.Now(),
		Payments:                   repository.states(),
	}, nil
}

type commandServices struct {
	repository *memoryRepository
}

func (s commandServices) execute(
	ctx context.Context,
	command replaydomain.Command,
) error {
	switch command.Type {
	case replaydomain.CommandCreatePayment:
		_, err := paymentapplication.NewCreatePayment(s.repository).Execute(
			ctx,
			paymentapplication.CreatePaymentCommand{
				ID:       command.PaymentID,
				Amount:   command.Amount.Amount(),
				Currency: command.Amount.Currency(),
			},
		)
		return err
	case replaydomain.CommandAuthorize:
		_, err := paymentapplication.NewAuthorizePayment(s.repository).Execute(
			ctx,
			paymentapplication.AuthorizePaymentCommand{PaymentID: command.PaymentID},
		)
		return err
	case replaydomain.CommandCapture:
		_, err := paymentapplication.NewCapturePayment(s.repository).Execute(
			ctx,
			paymentapplication.CapturePaymentCommand{
				PaymentID: command.PaymentID,
				Amount:    command.Amount.Amount(),
				Currency:  command.Amount.Currency(),
			},
		)
		return err
	case replaydomain.CommandRefund:
		_, err := paymentapplication.NewRefundPayment(s.repository).Execute(
			ctx,
			paymentapplication.RefundPaymentCommand{
				PaymentID: command.PaymentID,
				Amount:    command.Amount.Amount(),
				Currency:  command.Amount.Currency(),
			},
		)
		return err
	case replaydomain.CommandCancel:
		_, err := paymentapplication.NewCancelPayment(s.repository).Execute(
			ctx,
			paymentapplication.CancelPaymentCommand{PaymentID: command.PaymentID},
		)
		return err
	default:
		return replaydomain.ErrInvalidCommand
	}
}

type memoryRepository struct {
	payments map[paymentdomain.ID]*paymentdomain.Payment
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		payments: make(map[paymentdomain.ID]*paymentdomain.Payment),
	}
}

func (r *memoryRepository) Save(
	ctx context.Context,
	payment *paymentdomain.Payment,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	current, exists := r.payments[payment.ID()]
	if exists && current != payment {
		return fmt.Errorf("%w: %s", ErrPaymentAlreadyExists, payment.ID())
	}

	r.payments[payment.ID()] = payment
	return nil
}

func (r *memoryRepository) FindByID(
	ctx context.Context,
	id paymentdomain.ID,
) (*paymentdomain.Payment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	payment, exists := r.payments[id]
	if !exists {
		return nil, paymentapplication.ErrPaymentNotFound
	}

	return payment, nil
}

func (r *memoryRepository) states() []paymentdomain.PaymentState {
	ids := make([]paymentdomain.ID, 0, len(r.payments))
	for id := range r.payments {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	states := make([]paymentdomain.PaymentState, 0, len(ids))
	for _, id := range ids {
		payment := r.payments[id]
		states = append(states, paymentdomain.PaymentState{
			ID:               payment.ID(),
			Amount:           payment.Amount(),
			Status:           payment.Status(),
			AuthorizedAmount: payment.AuthorizedAmount().Amount(),
			CapturedAmount:   payment.CapturedAmount().Amount(),
			RefundedAmount:   payment.RefundedAmount().Amount(),
			Version:          payment.Version(),
		})
	}

	return states
}

package application

import (
	"context"
	"fmt"
	"sort"
	"time"

	paymentapplication "proxynth/payment-sandbox/internal/payment/application"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	"proxynth/payment-sandbox/internal/platform/clock"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
	replaydomain "proxynth/payment-sandbox/internal/replay/domain"
)

// Result contains the deterministic state produced by one scenario execution.
type Result struct {
	ScenarioID                 replaydomain.ScenarioID
	Provider                   replaydomain.ProviderConfiguration
	DeterministicConfiguration replaydomain.DeterministicConfiguration
	CurrentVirtualTime         time.Time
	Payments                   []paymentdomain.PaymentState
	AsyncOperations            []providerdomain.AsyncOperation
}

// ProviderRegistry resolves the provider selected by a scenario.
type ProviderRegistry interface {
	Resolve(providerdomain.ProviderID) (providerdomain.Provider, error)
}

// Runner executes one scenario against the production payment application
// services and an execution-scoped in-memory repository.
type Runner struct {
	providers ProviderRegistry
}

func NewRunner(providers ProviderRegistry) *Runner {
	return &Runner{providers: providers}
}

func (r *Runner) Run(ctx context.Context, scenario replaydomain.Scenario) (Result, error) {
	if err := scenario.Validate(); err != nil {
		return Result{}, err
	}

	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if r.providers == nil {
		return Result{}, ErrNilProviderRegistry
	}

	provider, err := r.providers.Resolve(scenario.Provider.ID)
	if err != nil {
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

	services := commandServices{repository: repository, provider: provider}
	services.now = virtualClock.Now()
	asyncOperations := make([]providerdomain.AsyncOperation, 0)
	for index, command := range scenario.Commands {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}

		operations, err := services.execute(ctx, command)
		if err != nil {
			return Result{}, fmt.Errorf(
				"execute scenario command %d (%s): %w",
				index,
				command.Type,
				err,
			)
		}
		asyncOperations = appendAsyncOperations(asyncOperations, operations...)
	}

	return Result{
		ScenarioID:                 scenario.ID,
		Provider:                   scenario.Provider,
		DeterministicConfiguration: scenario.DeterministicConfiguration,
		CurrentVirtualTime:         virtualClock.Now(),
		Payments:                   repository.states(),
		AsyncOperations:            asyncOperations,
	}, nil
}

type commandServices struct {
	repository *memoryRepository
	provider   providerdomain.Provider
	now        time.Time
}

func (s commandServices) execute(
	ctx context.Context,
	command replaydomain.Command,
) ([]providerdomain.AsyncOperation, error) {
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
		return nil, err
	case replaydomain.CommandAuthorize:
		payment, err := s.repository.FindByID(ctx, command.PaymentID)
		if err != nil {
			return nil, err
		}
		result, err := s.authorize(ctx, payment)
		if err != nil {
			return nil, err
		}
		_, err = paymentapplication.NewAuthorizePayment(s.repository).Execute(
			ctx,
			paymentapplication.AuthorizePaymentCommand{PaymentID: command.PaymentID},
		)
		return result.AsyncOperations, err
	case replaydomain.CommandCapture:
		payment, err := s.repository.FindByID(ctx, command.PaymentID)
		if err != nil {
			return nil, err
		}
		result, err := s.capture(ctx, payment, command.Amount)
		if err != nil {
			return nil, err
		}
		_, err = paymentapplication.NewCapturePayment(s.repository).Execute(
			ctx,
			paymentapplication.CapturePaymentCommand{
				PaymentID: command.PaymentID,
				Amount:    command.Amount.Amount(),
				Currency:  command.Amount.Currency(),
			},
		)
		return result.AsyncOperations, err
	case replaydomain.CommandRefund:
		payment, err := s.repository.FindByID(ctx, command.PaymentID)
		if err != nil {
			return nil, err
		}
		result, err := s.refund(ctx, payment, command.Amount)
		if err != nil {
			return nil, err
		}
		_, err = paymentapplication.NewRefundPayment(s.repository).Execute(
			ctx,
			paymentapplication.RefundPaymentCommand{
				PaymentID: command.PaymentID,
				Amount:    command.Amount.Amount(),
				Currency:  command.Amount.Currency(),
			},
		)
		return result.AsyncOperations, err
	case replaydomain.CommandCancel:
		payment, err := s.repository.FindByID(ctx, command.PaymentID)
		if err != nil {
			return nil, err
		}
		result, err := s.cancel(ctx, payment)
		if err != nil {
			return nil, err
		}
		_, err = paymentapplication.NewCancelPayment(s.repository).Execute(
			ctx,
			paymentapplication.CancelPaymentCommand{PaymentID: command.PaymentID},
		)
		return result.AsyncOperations, err
	default:
		return nil, replaydomain.ErrInvalidCommand
	}
}

func (s commandServices) authorize(ctx context.Context, payment *paymentdomain.Payment) (providerResult providerdomain.OperationResult, err error) {
	return validateProviderResult(s.provider.Authorize(ctx, providerdomain.AuthorizeRequest{
		Payment: paymentSnapshot(payment),
		At:      s.now,
	}))
}

func (s commandServices) capture(ctx context.Context, payment *paymentdomain.Payment, amount paymentdomain.Money) (providerResult providerdomain.OperationResult, err error) {
	return validateProviderResult(s.provider.Capture(ctx, providerdomain.CaptureRequest{
		Payment: paymentSnapshot(payment),
		Amount:  amount,
		At:      s.now,
	}))
}

func (s commandServices) refund(ctx context.Context, payment *paymentdomain.Payment, amount paymentdomain.Money) (providerResult providerdomain.OperationResult, err error) {
	return validateProviderResult(s.provider.Refund(ctx, providerdomain.RefundRequest{
		Payment: paymentSnapshot(payment),
		Amount:  amount,
		At:      s.now,
	}))
}

func (s commandServices) cancel(ctx context.Context, payment *paymentdomain.Payment) (providerResult providerdomain.OperationResult, err error) {
	return validateProviderResult(s.provider.Cancel(ctx, providerdomain.CancelRequest{
		Payment: paymentSnapshot(payment),
		At:      s.now,
	}))
}

func paymentSnapshot(payment *paymentdomain.Payment) providerdomain.PaymentSnapshot {
	return providerdomain.PaymentSnapshot{
		ID:      payment.ID(),
		Amount:  payment.Amount(),
		Status:  payment.Status(),
		Version: payment.Version(),
	}
}

func validateProviderResult(result providerdomain.OperationResult, err error) (providerdomain.OperationResult, error) {
	if err != nil {
		return providerdomain.OperationResult{}, err
	}

	if err := result.Validate(); err != nil {
		return providerdomain.OperationResult{}, err
	}

	return result, nil
}

func appendAsyncOperations(
	operations []providerdomain.AsyncOperation,
	additional ...providerdomain.AsyncOperation,
) []providerdomain.AsyncOperation {
	for _, operation := range additional {
		operation.Payload = append([]byte(nil), operation.Payload...)
		operations = append(operations, operation)
	}

	return operations
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

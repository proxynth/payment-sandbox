package deterministic

import (
	"context"
	"fmt"
	"time"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
)

// Provider is a stateless provider implementation for deterministic success
// scenarios. Concrete providers embed it and provide their own identity.
type Provider struct {
	identity providerdomain.ProviderIdentity
	profile  string
	seed     uint64
}

func New(id providerdomain.ProviderID) *Provider {
	return NewWithProfile(id, "success")
}

func NewWithProfile(id providerdomain.ProviderID, profile string) *Provider {
	return &Provider{identity: providerdomain.ProviderIdentity{ID: id}, profile: profile}
}

func (p Provider) Identity() providerdomain.ProviderIdentity {
	return p.identity
}

func (p Provider) Authorize(_ context.Context, request providerdomain.AuthorizeRequest) (providerdomain.OperationResult, error) {
	if err := validateSnapshot(request.Payment); err != nil {
		return providerdomain.OperationResult{}, err
	}

	return p.result("authorize", request.Payment.ID, request.At)
}

func (p Provider) Capture(_ context.Context, request providerdomain.CaptureRequest) (providerdomain.OperationResult, error) {
	if err := validateSnapshot(request.Payment); err != nil {
		return providerdomain.OperationResult{}, err
	}

	if err := validateOperationAmount(request.Payment, request.Amount); err != nil {
		return providerdomain.OperationResult{}, err
	}

	return p.result("capture", request.Payment.ID, request.At)
}

func (p Provider) Refund(_ context.Context, request providerdomain.RefundRequest) (providerdomain.OperationResult, error) {
	if err := validateSnapshot(request.Payment); err != nil {
		return providerdomain.OperationResult{}, err
	}

	if err := validateOperationAmount(request.Payment, request.Amount); err != nil {
		return providerdomain.OperationResult{}, err
	}

	return p.result("refund", request.Payment.ID, request.At)
}

func (p Provider) Cancel(_ context.Context, request providerdomain.CancelRequest) (providerdomain.OperationResult, error) {
	if err := validateSnapshot(request.Payment); err != nil {
		return providerdomain.OperationResult{}, err
	}

	return p.result("cancel", request.Payment.ID, request.At)
}

func (p Provider) Configure(profile string, seed uint64) (providerdomain.Provider, error) {
	if profile == "" {
		profile = "success"
	}
	return &Provider{identity: p.identity, profile: profile, seed: seed}, nil
}

func (p Provider) ExecuteAsync(_ context.Context, operation providerdomain.AsyncOperation) (providerdomain.OperationResult, error) {
	if err := operation.Validate(); err != nil {
		return providerdomain.OperationResult{}, err
	}
	return successfulResult(operation.Type, operation.PaymentID, p.identity.ID), nil
}

func (p Provider) result(operation string, paymentID paymentdomain.ID, at time.Time) (providerdomain.OperationResult, error) {
	switch p.profile {
	case "seeded":
		switch p.seed % 3 {
		case 1:
			if operation == "authorize" {
				return failedResult(operation, paymentID, p.identity.ID), nil
			}
		case 2:
			if operation == "authorize" {
				return pendingResult(operation, paymentID, p.identity.ID, at, p.seed%5+1), nil
			}
		}
	case "fail_authorize":
		if operation == "authorize" {
			return failedResult(operation, paymentID, p.identity.ID), nil
		}
	case "pending_authorize":
		if operation == "authorize" {
			return providerdomain.OperationResult{
				Outcome: providerdomain.OutcomePending,
				AsyncOperations: []providerdomain.AsyncOperation{{
					ID:          fmt.Sprintf("%s:%s:async", p.identity.ID, paymentID),
					PaymentID:   paymentID,
					Type:        operation,
					ScheduledAt: at.Add(time.Minute),
				}},
			}, nil
		}
	}

	return successfulResult(operation, paymentID, p.identity.ID), nil
}

func pendingResult(operation string, paymentID paymentdomain.ID, providerID providerdomain.ProviderID, at time.Time, delay uint64) providerdomain.OperationResult {
	return providerdomain.OperationResult{
		Outcome: providerdomain.OutcomePending,
		AsyncOperations: []providerdomain.AsyncOperation{{
			ID:          fmt.Sprintf("%s:%s:async", providerID, paymentID),
			PaymentID:   paymentID,
			Type:        operation,
			ScheduledAt: at.Add(time.Duration(delay) * time.Minute),
		}},
	}
}

func successfulResult(operation string, paymentID paymentdomain.ID, providerID providerdomain.ProviderID) providerdomain.OperationResult {
	return providerdomain.OperationResult{
		Outcome:           providerdomain.OutcomeSucceeded,
		ProviderReference: fmt.Sprintf("%s:%s:%s", providerID, operation, paymentID),
	}
}

func failedResult(operation string, paymentID paymentdomain.ID, providerID providerdomain.ProviderID) providerdomain.OperationResult {
	return providerdomain.OperationResult{
		Outcome:           providerdomain.OutcomeFailed,
		ProviderReference: fmt.Sprintf("%s:%s:%s:failed", providerID, operation, paymentID),
	}
}

func validateSnapshot(snapshot providerdomain.PaymentSnapshot) error {
	if err := snapshot.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidRequest, err)
	}

	if _, err := paymentdomain.NewMoney(snapshot.Amount.Amount(), snapshot.Amount.Currency()); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidRequest, err)
	}

	return nil
}

func validateOperationAmount(snapshot providerdomain.PaymentSnapshot, amount paymentdomain.Money) error {
	if amount.Amount() <= 0 || amount.Currency() != snapshot.Amount.Currency() {
		return ErrInvalidOperationAmount
	}

	if _, err := paymentdomain.NewMoney(amount.Amount(), amount.Currency()); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidOperationAmount, err)
	}

	return nil
}

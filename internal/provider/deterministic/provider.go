package deterministic

import (
	"context"
	"fmt"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
)

// Provider is a stateless provider implementation for deterministic success
// scenarios. Concrete providers embed it and provide their own identity.
type Provider struct {
	identity providerdomain.ProviderIdentity
}

func New(id providerdomain.ProviderID) *Provider {
	return &Provider{identity: providerdomain.ProviderIdentity{ID: id}}
}

func (p Provider) Identity() providerdomain.ProviderIdentity {
	return p.identity
}

func (p Provider) Authorize(_ context.Context, request providerdomain.AuthorizeRequest) (providerdomain.OperationResult, error) {
	if err := validateSnapshot(request.Payment); err != nil {
		return providerdomain.OperationResult{}, err
	}

	return successfulResult("authorize", request.Payment.ID, p.identity.ID), nil
}

func (p Provider) Capture(_ context.Context, request providerdomain.CaptureRequest) (providerdomain.OperationResult, error) {
	if err := validateSnapshot(request.Payment); err != nil {
		return providerdomain.OperationResult{}, err
	}

	if err := validateOperationAmount(request.Payment, request.Amount); err != nil {
		return providerdomain.OperationResult{}, err
	}

	return successfulResult("capture", request.Payment.ID, p.identity.ID), nil
}

func (p Provider) Refund(_ context.Context, request providerdomain.RefundRequest) (providerdomain.OperationResult, error) {
	if err := validateSnapshot(request.Payment); err != nil {
		return providerdomain.OperationResult{}, err
	}

	if err := validateOperationAmount(request.Payment, request.Amount); err != nil {
		return providerdomain.OperationResult{}, err
	}

	return successfulResult("refund", request.Payment.ID, p.identity.ID), nil
}

func (p Provider) Cancel(_ context.Context, request providerdomain.CancelRequest) (providerdomain.OperationResult, error) {
	if err := validateSnapshot(request.Payment); err != nil {
		return providerdomain.OperationResult{}, err
	}

	return successfulResult("cancel", request.Payment.ID, p.identity.ID), nil
}

func successfulResult(operation string, paymentID paymentdomain.ID, providerID providerdomain.ProviderID) providerdomain.OperationResult {
	return providerdomain.OperationResult{
		Outcome:           providerdomain.OutcomeSucceeded,
		ProviderReference: fmt.Sprintf("%s:%s:%s", providerID, operation, paymentID),
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

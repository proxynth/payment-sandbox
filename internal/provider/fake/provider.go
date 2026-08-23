package fake

import (
	"context"
	"fmt"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
)

const providerID providerdomain.ProviderID = "fake"

// Provider is a deterministic, stateless provider implementation for local
// simulations and contract tests.
type Provider struct{}

var _ providerdomain.Provider = (*Provider)(nil)

func New() *Provider {
	return &Provider{}
}

func (Provider) Identity() providerdomain.ProviderIdentity {
	return providerdomain.ProviderIdentity{ID: providerID}
}

func (Provider) Authorize(
	_ context.Context,
	request providerdomain.AuthorizeRequest,
) (providerdomain.OperationResult, error) {
	if err := validateSnapshot(request.Payment); err != nil {
		return providerdomain.OperationResult{}, err
	}

	return successfulResult("authorize", request.Payment.ID), nil
}

func (Provider) Capture(
	_ context.Context,
	request providerdomain.CaptureRequest,
) (providerdomain.OperationResult, error) {
	if err := validateSnapshot(request.Payment); err != nil {
		return providerdomain.OperationResult{}, err
	}

	if err := validateOperationAmount(request.Payment, request.Amount); err != nil {
		return providerdomain.OperationResult{}, err
	}

	return successfulResult("capture", request.Payment.ID), nil
}

func (Provider) Refund(
	_ context.Context,
	request providerdomain.RefundRequest,
) (providerdomain.OperationResult, error) {
	if err := validateSnapshot(request.Payment); err != nil {
		return providerdomain.OperationResult{}, err
	}

	if err := validateOperationAmount(request.Payment, request.Amount); err != nil {
		return providerdomain.OperationResult{}, err
	}

	return successfulResult("refund", request.Payment.ID), nil
}

func (Provider) Cancel(
	_ context.Context,
	request providerdomain.CancelRequest,
) (providerdomain.OperationResult, error) {
	if err := validateSnapshot(request.Payment); err != nil {
		return providerdomain.OperationResult{}, err
	}

	return successfulResult("cancel", request.Payment.ID), nil
}

func successfulResult(operation string, paymentID paymentdomain.ID) providerdomain.OperationResult {
	return providerdomain.OperationResult{
		Outcome:           providerdomain.OutcomeSucceeded,
		ProviderReference: fmt.Sprintf("fake:%s:%s", operation, paymentID),
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

func validateOperationAmount(
	snapshot providerdomain.PaymentSnapshot,
	amount paymentdomain.Money,
) error {
	if amount.Amount() <= 0 || amount.Currency() != snapshot.Amount.Currency() {
		return ErrInvalidOperationAmount
	}

	if _, err := paymentdomain.NewMoney(amount.Amount(), amount.Currency()); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidOperationAmount, err)
	}

	return nil
}

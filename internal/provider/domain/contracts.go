package domain

import (
	"context"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
)

// ProviderID identifies a provider without exposing provider-specific types to
// the payment domain.
type ProviderID string

func (id ProviderID) Valid() bool {
	return id != ""
}

// ProviderIdentity describes the stable metadata required to select a
// provider. Provider-specific configuration belongs to the implementation.
type ProviderIdentity struct {
	ID ProviderID
}

func (identity ProviderIdentity) Validate() error {
	if !identity.ID.Valid() {
		return ErrInvalidProviderID
	}

	return nil
}

// PaymentSnapshot is the canonical payment input shared by provider
// operations. Providers receive a value snapshot and cannot mutate the
// payment aggregate or its persisted state through this contract.
type PaymentSnapshot struct {
	ID      paymentdomain.ID
	Amount  paymentdomain.Money
	Status  paymentdomain.Status
	Version uint64
}

func (snapshot PaymentSnapshot) Validate() error {
	if snapshot.ID == "" || !snapshot.Status.Valid() || snapshot.Version == 0 {
		return ErrInvalidPaymentSnapshot
	}

	if snapshot.Amount.Amount() <= 0 || snapshot.Amount.Currency() == "" {
		return ErrInvalidPaymentSnapshot
	}

	return nil
}

type AuthorizeRequest struct {
	Payment PaymentSnapshot
}

type CaptureRequest struct {
	Payment PaymentSnapshot
	Amount  paymentdomain.Money
}

type RefundRequest struct {
	Payment PaymentSnapshot
	Amount  paymentdomain.Money
}

type CancelRequest struct {
	Payment PaymentSnapshot
}

// OperationOutcome describes the business outcome returned by a provider.
// Failed is a business result; transport and implementation failures are
// returned as Go errors.
type OperationOutcome string

const (
	OutcomeSucceeded OperationOutcome = "succeeded"
	OutcomePending   OperationOutcome = "pending"
	OutcomeFailed    OperationOutcome = "failed"
)

func (outcome OperationOutcome) Valid() bool {
	switch outcome {
	case OutcomeSucceeded, OutcomePending, OutcomeFailed:
		return true
	default:
		return false
	}
}

type OperationResult struct {
	Outcome           OperationOutcome
	ProviderReference string
}

func (result OperationResult) Validate() error {
	if !result.Outcome.Valid() {
		return ErrInvalidOperationResult
	}

	return nil
}

type Authorizer interface {
	Authorize(context.Context, AuthorizeRequest) (OperationResult, error)
}

type Capturer interface {
	Capture(context.Context, CaptureRequest) (OperationResult, error)
}

type Refunder interface {
	Refund(context.Context, RefundRequest) (OperationResult, error)
}

type Canceller interface {
	Cancel(context.Context, CancelRequest) (OperationResult, error)
}

// Provider is the complete provider contract. Individual capabilities remain
// separate so future providers can advertise a narrower supported surface.
type Provider interface {
	Identity() ProviderIdentity
	Authorizer
	Capturer
	Refunder
	Canceller
}

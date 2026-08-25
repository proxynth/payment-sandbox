package domain

import (
	"context"
	"time"

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
	At      time.Time
}

type CaptureRequest struct {
	Payment PaymentSnapshot
	Amount  paymentdomain.Money
	At      time.Time
}

type RefundRequest struct {
	Payment PaymentSnapshot
	Amount  paymentdomain.Money
	At      time.Time
}

type CancelRequest struct {
	Payment PaymentSnapshot
	At      time.Time
}

// AsyncOperation describes deterministic future work requested by a provider.
// The runtime is responsible for persisting and executing it.
type AsyncOperation struct {
	ID          string
	PaymentID   paymentdomain.ID
	Type        string
	Payload     []byte
	ScheduledAt time.Time
}

func (operation AsyncOperation) Validate() error {
	if operation.ID == "" || operation.PaymentID == "" || operation.Type == "" || operation.ScheduledAt.IsZero() {
		return ErrInvalidAsyncOperation
	}

	return nil
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
	AsyncOperations   []AsyncOperation
}

// ConfigurableProvider creates an execution-scoped provider configured for a
// deterministic scenario. The configuration remains opaque to the replay
// core; only the provider interprets it.
type ConfigurableProvider interface {
	Configure(string, uint64) (Provider, error)
}

// AsyncExecutor resumes provider-owned asynchronous work once the runtime has
// reached its scheduled virtual time.
type AsyncExecutor interface {
	ExecuteAsync(context.Context, AsyncOperation) (OperationResult, error)
}

func (result OperationResult) Validate() error {
	if !result.Outcome.Valid() {
		return ErrInvalidOperationResult
	}

	seen := make(map[string]struct{}, len(result.AsyncOperations))
	for _, operation := range result.AsyncOperations {
		if err := operation.Validate(); err != nil {
			return err
		}
		if _, exists := seen[operation.ID]; exists {
			return ErrInvalidAsyncOperation
		}
		seen[operation.ID] = struct{}{}
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

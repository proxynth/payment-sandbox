package domain

import (
	"context"
	"errors"
	"testing"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
)

type fakeProvider struct{}

var _ Provider = (*fakeProvider)(nil)

func (fakeProvider) Identity() ProviderIdentity {
	return ProviderIdentity{ID: "fake"}
}

func (fakeProvider) Authorize(context.Context, AuthorizeRequest) (OperationResult, error) {
	return OperationResult{Outcome: OutcomeSucceeded}, nil
}

func (fakeProvider) Capture(context.Context, CaptureRequest) (OperationResult, error) {
	return OperationResult{Outcome: OutcomeSucceeded}, nil
}

func (fakeProvider) Refund(context.Context, RefundRequest) (OperationResult, error) {
	return OperationResult{Outcome: OutcomeSucceeded}, nil
}

func (fakeProvider) Cancel(context.Context, CancelRequest) (OperationResult, error) {
	return OperationResult{Outcome: OutcomeSucceeded}, nil
}

func TestProviderContractCanBeImplementedWithoutProviderTypes(t *testing.T) {
	provider := fakeProvider{}

	if err := provider.Identity().Validate(); err != nil {
		t.Fatalf("provider identity should be valid: %v", err)
	}

	result, err := provider.Authorize(context.Background(), AuthorizeRequest{})
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}

	if err := result.Validate(); err != nil {
		t.Fatalf("provider result should be valid: %v", err)
	}
}

func TestProviderIdentity_RejectsEmptyID(t *testing.T) {
	err := (ProviderIdentity{}).Validate()

	if !errors.Is(err, ErrInvalidProviderID) {
		t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidProviderID)
	}
}

func TestPaymentSnapshot_RejectsInvalidMetadata(t *testing.T) {
	tests := []PaymentSnapshot{
		{Amount: mustMoney(t, 1000, "EUR"), Status: paymentdomain.StatusPending, Version: 1},
		{ID: "payment-1", Amount: mustMoney(t, 1000, "EUR"), Status: paymentdomain.Status("unknown"), Version: 1},
		{ID: "payment-1", Amount: mustMoney(t, 1000, "EUR"), Status: paymentdomain.StatusPending},
	}

	for _, snapshot := range tests {
		if err := snapshot.Validate(); !errors.Is(err, ErrInvalidPaymentSnapshot) {
			t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidPaymentSnapshot)
		}
	}
}

func TestOperationResult_RejectsUnknownOutcome(t *testing.T) {
	err := (OperationResult{Outcome: "unknown"}).Validate()

	if !errors.Is(err, ErrInvalidOperationResult) {
		t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidOperationResult)
	}
}

func mustMoney(t *testing.T, amount int64, currency paymentdomain.Currency) paymentdomain.Money {
	t.Helper()

	money, err := paymentdomain.NewMoney(amount, currency)
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}

	return money
}

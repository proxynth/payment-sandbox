package stripe

import (
	"context"
	"errors"
	"testing"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
)

func TestProvider_ImplementsContract(t *testing.T) {
	var provider providerdomain.Provider = New()

	if got := provider.Identity().ID; got != providerID {
		t.Fatalf("Identity().ID = %q, want %q", got, providerID)
	}
}

func TestProvider_CanBeRegisteredAndResolved(t *testing.T) {
	registry := providerdomain.NewRegistry()
	provider := New()

	if err := registry.Register(provider); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	resolved, err := registry.Resolve(providerID)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if resolved != provider {
		t.Fatalf("Resolve() returned a different provider")
	}
}

func TestProvider_OperationsReturnDeterministicSuccess(t *testing.T) {
	provider := New()
	snapshot := validSnapshot(t)
	amount := mustMoney(t, 4000, "EUR")

	tests := []struct {
		name string
		call func() (providerdomain.OperationResult, error)
		want string
	}{
		{
			name: "authorize",
			call: func() (providerdomain.OperationResult, error) {
				return provider.Authorize(context.Background(), providerdomain.AuthorizeRequest{Payment: snapshot})
			},
			want: "stripe:authorize:payment-1",
		},
		{
			name: "capture",
			call: func() (providerdomain.OperationResult, error) {
				return provider.Capture(context.Background(), providerdomain.CaptureRequest{Payment: snapshot, Amount: amount})
			},
			want: "stripe:capture:payment-1",
		},
		{
			name: "refund",
			call: func() (providerdomain.OperationResult, error) {
				return provider.Refund(context.Background(), providerdomain.RefundRequest{Payment: snapshot, Amount: amount})
			},
			want: "stripe:refund:payment-1",
		},
		{
			name: "cancel",
			call: func() (providerdomain.OperationResult, error) {
				return provider.Cancel(context.Background(), providerdomain.CancelRequest{Payment: snapshot})
			},
			want: "stripe:cancel:payment-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, err := tt.call()
			if err != nil {
				t.Fatalf("first call error = %v", err)
			}

			second, err := tt.call()
			if err != nil {
				t.Fatalf("second call error = %v", err)
			}

			if first != second {
				t.Fatalf("results differ: first = %+v, second = %+v", first, second)
			}

			if first.Outcome != providerdomain.OutcomeSucceeded {
				t.Fatalf("Outcome = %q, want %q", first.Outcome, providerdomain.OutcomeSucceeded)
			}

			if first.ProviderReference != tt.want {
				t.Fatalf("ProviderReference = %q, want %q", first.ProviderReference, tt.want)
			}
		})
	}
}

func TestProvider_RejectsInvalidSnapshot(t *testing.T) {
	_, err := New().Authorize(context.Background(), providerdomain.AuthorizeRequest{})

	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Authorize() error = %v, want %v", err, ErrInvalidRequest)
	}
}

func TestProvider_RejectsInvalidOperationAmount(t *testing.T) {
	snapshot := validSnapshot(t)
	amount := mustMoney(t, 0, "EUR")

	_, err := New().Capture(context.Background(), providerdomain.CaptureRequest{
		Payment: snapshot,
		Amount:  amount,
	})

	if !errors.Is(err, ErrInvalidOperationAmount) {
		t.Fatalf("Capture() error = %v, want %v", err, ErrInvalidOperationAmount)
	}
}

func TestProvider_DoesNotMutateRequest(t *testing.T) {
	provider := New()
	snapshot := validSnapshot(t)
	request := providerdomain.CaptureRequest{
		Payment: snapshot,
		Amount:  mustMoney(t, 4000, "EUR"),
	}

	if _, err := provider.Capture(context.Background(), request); err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	if request.Payment != snapshot {
		t.Fatalf("provider mutated the payment snapshot")
	}
}

func validSnapshot(t *testing.T) providerdomain.PaymentSnapshot {
	t.Helper()

	return providerdomain.PaymentSnapshot{
		ID:      "payment-1",
		Amount:  mustMoney(t, 10000, "EUR"),
		Status:  paymentdomain.StatusPending,
		Version: 1,
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

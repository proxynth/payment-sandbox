package deterministic

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
)

func TestProvider_ImplementsContract(t *testing.T) {
	var provider providerdomain.Provider = New("test")

	if got := provider.Identity().ID; got != "test" {
		t.Fatalf("Identity().ID = %q, want %q", got, "test")
	}
}

func TestProvider_OperationsReturnDeterministicSuccess(t *testing.T) {
	provider := New("test")
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
			want: "test:authorize:payment-1",
		},
		{
			name: "capture",
			call: func() (providerdomain.OperationResult, error) {
				return provider.Capture(context.Background(), providerdomain.CaptureRequest{Payment: snapshot, Amount: amount})
			},
			want: "test:capture:payment-1",
		},
		{
			name: "refund",
			call: func() (providerdomain.OperationResult, error) {
				return provider.Refund(context.Background(), providerdomain.RefundRequest{Payment: snapshot, Amount: amount})
			},
			want: "test:refund:payment-1",
		},
		{
			name: "cancel",
			call: func() (providerdomain.OperationResult, error) {
				return provider.Cancel(context.Background(), providerdomain.CancelRequest{Payment: snapshot})
			},
			want: "test:cancel:payment-1",
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

			if !reflect.DeepEqual(first, second) {
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

func TestProvider_UsesIdentityInReference(t *testing.T) {
	snapshot := validSnapshot(t)

	first, err := New("first").Authorize(context.Background(), providerdomain.AuthorizeRequest{Payment: snapshot})
	if err != nil {
		t.Fatalf("first Authorize() error = %v", err)
	}

	second, err := New("second").Authorize(context.Background(), providerdomain.AuthorizeRequest{Payment: snapshot})
	if err != nil {
		t.Fatalf("second Authorize() error = %v", err)
	}

	if first.ProviderReference == second.ProviderReference {
		t.Fatalf("provider references should include provider identity")
	}
}

func TestProvider_RejectsInvalidSnapshot(t *testing.T) {
	_, err := New("test").Authorize(context.Background(), providerdomain.AuthorizeRequest{})

	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Authorize() error = %v, want %v", err, ErrInvalidRequest)
	}
}

func TestProvider_SeededProfileUsesSeedDeterministically(t *testing.T) {
	snapshot := validSnapshot(t)
	at := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	want := map[uint64]providerdomain.OperationOutcome{
		0: providerdomain.OutcomeSucceeded,
		1: providerdomain.OutcomeFailed,
		2: providerdomain.OutcomePending,
	}
	for seed, expected := range want {
		configured, err := New("test").Configure("seeded", seed)
		if err != nil {
			t.Fatalf("Configure(%d) error = %v", seed, err)
		}
		first, err := configured.Authorize(context.Background(), providerdomain.AuthorizeRequest{Payment: snapshot, At: at})
		if err != nil {
			t.Fatalf("Authorize(%d) error = %v", seed, err)
		}
		second, err := configured.Authorize(context.Background(), providerdomain.AuthorizeRequest{Payment: snapshot, At: at})
		if err != nil {
			t.Fatalf("second Authorize(%d) error = %v", seed, err)
		}
		if first.Outcome != expected || !reflect.DeepEqual(first, second) {
			t.Fatalf("seed %d results = %+v and %+v, want deterministic %q", seed, first, second, expected)
		}
	}
}

func TestProvider_RejectsInvalidOperationAmount(t *testing.T) {
	snapshot := validSnapshot(t)
	amount := mustMoney(t, 0, "EUR")

	_, err := New("test").Capture(context.Background(), providerdomain.CaptureRequest{
		Payment: snapshot,
		Amount:  amount,
	})

	if !errors.Is(err, ErrInvalidOperationAmount) {
		t.Fatalf("Capture() error = %v, want %v", err, ErrInvalidOperationAmount)
	}
}

func TestProvider_DoesNotMutateRequest(t *testing.T) {
	snapshot := validSnapshot(t)
	request := providerdomain.CaptureRequest{
		Payment: snapshot,
		Amount:  mustMoney(t, 4000, "EUR"),
	}

	if _, err := New("test").Capture(context.Background(), request); err != nil {
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

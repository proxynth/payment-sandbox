package domain

import (
	"errors"
	"testing"
	"time"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
)

func TestNewScenario_PreservesBusinessInputs(t *testing.T) {
	location := time.FixedZone("CET", 3600)
	initialTime := time.Date(2026, 8, 24, 12, 0, 0, 0, location)
	initialPayments := []paymentdomain.PaymentState{
		{
			ID:      "payment-1",
			Amount:  mustMoney(t, 10000, "EUR"),
			Status:  paymentdomain.StatusPending,
			Version: 1,
		},
	}
	commands := []Command{
		{Type: CommandAuthorize, PaymentID: "payment-1"},
		{Type: CommandCapture, PaymentID: "payment-1", Amount: mustMoney(t, 10000, "EUR")},
		{Type: CommandRefund, PaymentID: "payment-1", Amount: mustMoney(t, 10000, "EUR")},
	}

	scenario, err := New(
		"scenario-1",
		initialPayments,
		commands,
		ProviderConfiguration{ID: "stripe"},
		initialTime,
		DeterministicConfiguration{Seed: 42},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if scenario.InitialVirtualTime.Location() != time.UTC {
		t.Fatalf("InitialVirtualTime location = %v, want UTC", scenario.InitialVirtualTime.Location())
	}

	if scenario.Commands[0].Type != CommandAuthorize ||
		scenario.Commands[1].Type != CommandCapture ||
		scenario.Commands[2].Type != CommandRefund {
		t.Fatalf("command order was not preserved: %+v", scenario.Commands)
	}

	if scenario.DeterministicConfiguration.Seed != 42 {
		t.Fatalf("Seed = %d, want 42", scenario.DeterministicConfiguration.Seed)
	}
}

func TestScenario_ValidationIsDeterministic(t *testing.T) {
	scenario := Scenario{
		ID:                 "scenario-1",
		Provider:           ProviderConfiguration{ID: "fake"},
		InitialVirtualTime: time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC),
		Commands:           []Command{{Type: CommandAuthorize, PaymentID: "payment-1"}},
	}

	first := scenario.Validate()
	second := scenario.Validate()

	if (first == nil) != (second == nil) {
		t.Fatalf("repeated validation differs: first = %v, second = %v", first, second)
	}

	if first != nil && first.Error() != second.Error() {
		t.Fatalf("repeated validation differs: first = %v, second = %v", first, second)
	}
}

func TestScenario_RejectsInvalidStructure(t *testing.T) {
	validTime := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	validPayment := paymentdomain.PaymentState{
		ID:      "payment-1",
		Amount:  mustMoney(t, 10000, "EUR"),
		Status:  paymentdomain.StatusPending,
		Version: 1,
	}

	tests := []struct {
		name     string
		scenario Scenario
		want     error
	}{
		{
			name: "empty id",
			scenario: Scenario{
				Provider:           ProviderConfiguration{ID: "fake"},
				InitialVirtualTime: validTime,
			},
			want: ErrInvalidScenarioID,
		},
		{
			name: "zero time",
			scenario: Scenario{
				ID:       "scenario-1",
				Provider: ProviderConfiguration{ID: "fake"},
			},
			want: ErrInvalidScenarioTime,
		},
		{
			name: "invalid provider",
			scenario: Scenario{
				ID:                 "scenario-1",
				Provider:           ProviderConfiguration{},
				InitialVirtualTime: validTime,
			},
			want: ErrInvalidProviderConfiguration,
		},
		{
			name: "duplicate initial payment",
			scenario: Scenario{
				ID:                 "scenario-1",
				Provider:           ProviderConfiguration{ID: "fake"},
				InitialVirtualTime: validTime,
				InitialPayments:    []paymentdomain.PaymentState{validPayment, validPayment},
			},
			want: ErrDuplicateInitialPayment,
		},
		{
			name: "unknown command",
			scenario: Scenario{
				ID:                 "scenario-1",
				Provider:           ProviderConfiguration{ID: "fake"},
				InitialVirtualTime: validTime,
				Commands:           []Command{{Type: "unknown", PaymentID: "payment-1"}},
			},
			want: ErrInvalidCommand,
		},
		{
			name: "missing command payment",
			scenario: Scenario{
				ID:                 "scenario-1",
				Provider:           ProviderConfiguration{ID: "fake"},
				InitialVirtualTime: validTime,
				Commands:           []Command{{Type: CommandAuthorize}},
			},
			want: ErrInvalidCommand,
		},
		{
			name: "invalid command amount",
			scenario: Scenario{
				ID:                 "scenario-1",
				Provider:           ProviderConfiguration{ID: "fake"},
				InitialVirtualTime: validTime,
				Commands:           []Command{{Type: CommandCapture, PaymentID: "payment-1"}},
			},
			want: ErrInvalidCommand,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.scenario.Validate(); !errors.Is(err, tt.want) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.want)
			}
		})
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

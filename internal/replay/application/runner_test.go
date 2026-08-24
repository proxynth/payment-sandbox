package application

import (
	"context"
	"errors"
	"testing"
	"time"

	paymentapplication "proxynth/payment-sandbox/internal/payment/application"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	replaydomain "proxynth/payment-sandbox/internal/replay/domain"
)

func TestRunner_ExecutesCommandsInOrder(t *testing.T) {
	amount := testMoney(t, 10000, "EUR")
	scenario := replaydomain.Scenario{
		ID:                 "scenario-1",
		Provider:           replaydomain.ProviderConfiguration{ID: "fake"},
		InitialVirtualTime: time.Date(2026, 8, 24, 12, 0, 0, 0, time.FixedZone("CET", 3600)),
		DeterministicConfiguration: replaydomain.DeterministicConfiguration{
			Seed: 42,
		},
		Commands: []replaydomain.Command{
			{Type: replaydomain.CommandCreatePayment, PaymentID: "payment-1", Amount: amount},
			{Type: replaydomain.CommandAuthorize, PaymentID: "payment-1"},
			{Type: replaydomain.CommandCapture, PaymentID: "payment-1", Amount: amount},
			{Type: replaydomain.CommandRefund, PaymentID: "payment-1", Amount: amount},
		},
	}

	result, err := NewRunner().Run(context.Background(), scenario)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.ScenarioID != scenario.ID {
		t.Errorf("ScenarioID = %q, want %q", result.ScenarioID, scenario.ID)
	}
	if result.Provider != scenario.Provider {
		t.Errorf("Provider = %+v, want %+v", result.Provider, scenario.Provider)
	}
	if result.DeterministicConfiguration != scenario.DeterministicConfiguration {
		t.Errorf("DeterministicConfiguration = %+v, want %+v", result.DeterministicConfiguration, scenario.DeterministicConfiguration)
	}
	if !result.CurrentVirtualTime.Equal(scenario.InitialVirtualTime.UTC()) {
		t.Errorf("CurrentVirtualTime = %v, want %v", result.CurrentVirtualTime, scenario.InitialVirtualTime.UTC())
	}

	if len(result.Payments) != 1 {
		t.Fatalf("Payments length = %d, want 1", len(result.Payments))
	}
	if result.Payments[0].Status != paymentdomain.StatusRefunded {
		t.Errorf("payment status = %q, want %q", result.Payments[0].Status, paymentdomain.StatusRefunded)
	}
	if result.Payments[0].Version != 4 {
		t.Errorf("payment version = %d, want 4", result.Payments[0].Version)
	}
}

func TestRunner_RestoresInitialState(t *testing.T) {
	initial := paymentdomain.PaymentState{
		ID:               "payment-1",
		Amount:           testMoney(t, 10000, "EUR"),
		Status:           paymentdomain.StatusAuthorized,
		AuthorizedAmount: 10000,
		Version:          2,
	}
	scenario := validScenario([]paymentdomain.PaymentState{initial}, []replaydomain.Command{
		{Type: replaydomain.CommandCapture, PaymentID: initial.ID, Amount: testMoney(t, 4000, "EUR")},
	})

	result, err := NewRunner().Run(context.Background(), scenario)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := result.Payments[0]; got.CapturedAmount != 4000 || got.Status != paymentdomain.StatusPartiallyCaptured {
		t.Fatalf("restored payment state = %+v, want partial capture", got)
	}
}

func TestRunner_StopsAtFirstCommandError(t *testing.T) {
	scenario := validScenario(nil, []replaydomain.Command{
		{Type: replaydomain.CommandAuthorize, PaymentID: "missing"},
		{Type: replaydomain.CommandCreatePayment, PaymentID: "should-not-exist", Amount: testMoney(t, 100, "EUR")},
	})

	_, err := NewRunner().Run(context.Background(), scenario)
	if !errors.Is(err, paymentapplication.ErrPaymentNotFound) {
		t.Fatalf("Run() error = %v, want %v", err, paymentapplication.ErrPaymentNotFound)
	}
}

func TestRunner_RejectsInvalidScenario(t *testing.T) {
	_, err := NewRunner().Run(context.Background(), replaydomain.Scenario{})
	if !errors.Is(err, replaydomain.ErrInvalidScenarioID) {
		t.Fatalf("Run() error = %v, want %v", err, replaydomain.ErrInvalidScenarioID)
	}
}

func TestRunner_RespectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewRunner().Run(ctx, validScenario(nil, nil))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want %v", err, context.Canceled)
	}
}

func validScenario(
	initialPayments []paymentdomain.PaymentState,
	commands []replaydomain.Command,
) replaydomain.Scenario {
	return replaydomain.Scenario{
		ID:                 "scenario-1",
		InitialPayments:    initialPayments,
		Commands:           commands,
		Provider:           replaydomain.ProviderConfiguration{ID: "fake"},
		InitialVirtualTime: time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC),
	}
}

func testMoney(t *testing.T, amount int64, currency paymentdomain.Currency) paymentdomain.Money {
	t.Helper()

	money, err := paymentdomain.NewMoney(amount, currency)
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}

	return money
}

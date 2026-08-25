package scenarios

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	adyenprovider "proxynth/payment-sandbox/internal/provider/adyen"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
	fakeprovider "proxynth/payment-sandbox/internal/provider/fake"
	stripeprovider "proxynth/payment-sandbox/internal/provider/stripe"
	replayapplication "proxynth/payment-sandbox/internal/replay/application"
	replaydomain "proxynth/payment-sandbox/internal/replay/domain"
)

func TestNewBuildsCanonicalProviderScenarios(t *testing.T) {
	amount := scenarioMoney(t)
	at := time.Date(2026, 8, 25, 12, 0, 0, 0, time.FixedZone("CET", 3600))

	scenario, err := New(PaymentLifecycle, "stripe", "stripe-lifecycle", at, amount, 42)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := scenario.Validate(); err != nil {
		t.Fatalf("Scenario.Validate() error = %v", err)
	}
	if scenario.Provider.ID != "stripe" || len(scenario.Commands) != 4 {
		t.Fatalf("scenario = %+v, want stripe lifecycle with four commands", scenario)
	}
	if scenario.Commands[0].Type != replaydomain.CommandCreatePayment || scenario.Commands[3].Type != replaydomain.CommandRefund {
		t.Fatalf("commands = %+v, want create ... refund", scenario.Commands)
	}
}

func TestNewRejectsUnknownScenarioName(t *testing.T) {
	_, err := New("unknown", "fake", "scenario-1", time.Now(), scenarioMoney(t), 1)
	if !errors.Is(err, ErrInvalidName) {
		t.Fatalf("New() error = %v, want %v", err, ErrInvalidName)
	}
}

func TestProviderScenariosExecuteIdenticallyAcrossProviders(t *testing.T) {
	providers := []struct {
		id       providerdomain.ProviderID
		provider providerdomain.Provider
	}{
		{id: "fake", provider: fakeprovider.New()},
		{id: "stripe", provider: stripeprovider.New()},
		{id: "adyen", provider: adyenprovider.New()},
	}

	for _, provider := range providers {
		t.Run(string(provider.id), func(t *testing.T) {
			registry := providerdomain.NewRegistry()
			if err := registry.Register(provider.provider); err != nil {
				t.Fatalf("Register() error = %v", err)
			}
			runner := replayapplication.NewRunner(registry)
			engine, err := replayapplication.NewReplayEngine(runner)
			if err != nil {
				t.Fatalf("NewReplayEngine() error = %v", err)
			}

			for _, name := range Names() {
				scenario, err := New(name, provider.id, replaydomain.ScenarioID(string(provider.id)+"-"+string(name)), time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC), scenarioMoney(t), 42)
				if err != nil {
					t.Fatalf("New(%q) error = %v", name, err)
				}
				first, err := engine.Replay(context.Background(), *scenario)
				if err != nil {
					t.Fatalf("first Replay(%q) error = %v", name, err)
				}
				second, err := engine.Replay(context.Background(), *scenario)
				if err != nil {
					t.Fatalf("second Replay(%q) error = %v", name, err)
				}
				if !reflect.DeepEqual(first, second) {
					t.Fatalf("replay results differ for %q: first = %+v, second = %+v", name, first, second)
				}
				wantStatus := paymentdomain.StatusRefunded
				if name == PaymentCancellation {
					wantStatus = paymentdomain.StatusCancelled
				}
				if len(first.Payments) != 1 || first.Payments[0].Status != wantStatus {
					t.Fatalf("payment result for %q = %+v, want status %q", name, first.Payments, wantStatus)
				}
			}
		})
	}
}

func TestProviderScenarioUsesScenarioPaymentIDAndSupportsPendingCompletion(t *testing.T) {
	provider := fakeprovider.New()
	registry := providerdomain.NewRegistry()
	if err := registry.Register(provider); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	runner := replayapplication.NewRunner(registry)
	amount := scenarioMoney(t)
	at := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	scenario, err := NewWithProfile(PaymentLifecycle, "fake", "pending_authorize", "pending-scenario", at, amount, 42)
	if err != nil {
		t.Fatalf("NewWithProfile() error = %v", err)
	}
	scenario.Commands = []replaydomain.Command{
		scenario.Commands[0],
		scenario.Commands[1],
		{Type: replaydomain.CommandAdvanceTime, Duration: time.Minute},
		{Type: replaydomain.CommandExecuteAsync, OperationID: "fake:pending-scenario-payment:async"},
	}
	result, err := runner.Run(context.Background(), *scenario)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Payments[0].ID != "pending-scenario-payment" || result.Payments[0].Status != paymentdomain.StatusAuthorized {
		t.Fatalf("payment = %+v, want scenario-owned ID and authorized status", result.Payments[0])
	}
	if len(result.AsyncOperations) != 1 {
		t.Fatalf("async operations = %d, want one", len(result.AsyncOperations))
	}
}

func TestProviderScenarioCanFailAuthorization(t *testing.T) {
	registry := providerdomain.NewRegistry()
	if err := registry.Register(fakeprovider.New()); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	scenario, err := NewWithProfile(PaymentLifecycle, "fake", "fail_authorize", "failed-scenario", time.Now(), scenarioMoney(t), 42)
	if err != nil {
		t.Fatalf("NewWithProfile() error = %v", err)
	}
	scenario.Commands = scenario.Commands[:2]
	result, err := replayapplication.NewRunner(registry).Run(context.Background(), *scenario)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Payments[0].Status != paymentdomain.StatusFailed {
		t.Fatalf("payment status = %q, want failed", result.Payments[0].Status)
	}
}

func scenarioMoney(t *testing.T) paymentdomain.Money {
	t.Helper()
	money, err := paymentdomain.NewMoney(1000, "EUR")
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}
	return money
}

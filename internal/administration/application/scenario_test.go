package application

import (
	"context"
	"errors"
	"testing"
	"time"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	replaydomain "proxynth/payment-sandbox/internal/replay/domain"
)

func TestScenarioInspectionReturnsValidatedScenario(t *testing.T) {
	scenario := validScenario(t)
	repository := &scenarioRepositoryFake{scenario: scenario}
	inspection, err := NewScenarioInspection(repository)
	if err != nil {
		t.Fatalf("NewScenarioInspection() error = %v", err)
	}

	got, err := inspection.Execute(context.Background(), scenario.ID)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got != scenario {
		t.Fatalf("Execute() scenario = %#v, want original scenario", got)
	}
	if repository.context == nil {
		t.Fatal("repository did not receive context")
	}
}

func TestScenarioInspectionMapsMissingScenario(t *testing.T) {
	inspection, err := NewScenarioInspection(&scenarioRepositoryFake{})
	if err != nil {
		t.Fatalf("NewScenarioInspection() error = %v", err)
	}

	_, err = inspection.Execute(context.Background(), "missing")
	if !errors.Is(err, ErrScenarioNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, ErrScenarioNotFound)
	}
}

func TestNewScenarioInspectionRejectsNilRepository(t *testing.T) {
	if _, err := NewScenarioInspection(nil); err == nil {
		t.Fatal("NewScenarioInspection() error = nil, want error")
	}
}

func TestScenarioInspectionRejectsInvalidStoredScenario(t *testing.T) {
	inspection, err := NewScenarioInspection(&scenarioRepositoryFake{
		scenario: &replaydomain.Scenario{ID: "invalid"},
	})
	if err != nil {
		t.Fatalf("NewScenarioInspection() error = %v", err)
	}

	_, err = inspection.Execute(context.Background(), "invalid")
	if !errors.Is(err, replaydomain.ErrInvalidScenarioTime) {
		t.Fatalf("Execute() error = %v, want %v", err, replaydomain.ErrInvalidScenarioTime)
	}
}

type scenarioRepositoryFake struct {
	scenario *replaydomain.Scenario
	err      error
	context  context.Context
}

func (r *scenarioRepositoryFake) FindByID(ctx context.Context, _ replaydomain.ScenarioID) (*replaydomain.Scenario, error) {
	r.context = ctx
	return r.scenario, r.err
}

func validScenario(t *testing.T) *replaydomain.Scenario {
	t.Helper()
	scenario, err := replaydomain.New(
		"scenario-inspection",
		nil,
		[]replaydomain.Command{{
			Type:      replaydomain.CommandCreatePayment,
			PaymentID: "payment-created",
			Amount:    scenarioMoney(t),
		}},
		replaydomain.ProviderConfiguration{ID: "stripe"},
		time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		replaydomain.DeterministicConfiguration{Seed: 42},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return scenario
}

func scenarioMoney(t *testing.T) paymentdomain.Money {
	t.Helper()
	money, err := paymentdomain.NewMoney(1000, "EUR")
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}
	return money
}

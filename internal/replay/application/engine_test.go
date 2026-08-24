package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	replaydomain "proxynth/payment-sandbox/internal/replay/domain"
)

func TestNewReplayEngine_RejectsNilRunner(t *testing.T) {
	_, err := NewReplayEngine(nil)
	if !errors.Is(err, ErrNilScenarioRunner) {
		t.Fatalf("NewReplayEngine() error = %v, want %v", err, ErrNilScenarioRunner)
	}
}

func TestReplayEngine_DelegatesValidScenario(t *testing.T) {
	scenario := engineTestScenario()
	want := Result{
		ScenarioID:         scenario.ID,
		Provider:           scenario.Provider,
		CurrentVirtualTime: scenario.InitialVirtualTime,
	}
	runner := &recordingRunner{result: want}
	engine, err := NewReplayEngine(runner)
	if err != nil {
		t.Fatalf("NewReplayEngine() error = %v", err)
	}

	got, err := engine.Replay(context.Background(), scenario)
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Replay() result = %+v, want %+v", got, want)
	}
	if !reflect.DeepEqual(runner.scenario, scenario) {
		t.Fatalf("runner scenario = %+v, want %+v", runner.scenario, scenario)
	}
}

func TestReplayEngine_RejectsInvalidScenarioBeforeDelegation(t *testing.T) {
	runner := &recordingRunner{}
	engine, err := NewReplayEngine(runner)
	if err != nil {
		t.Fatalf("NewReplayEngine() error = %v", err)
	}

	_, err = engine.Replay(context.Background(), replaydomain.Scenario{})
	if !errors.Is(err, replaydomain.ErrInvalidScenarioID) {
		t.Fatalf("Replay() error = %v, want %v", err, replaydomain.ErrInvalidScenarioID)
	}
	if runner.called {
		t.Fatal("runner was called for an invalid scenario")
	}
}

func TestReplayEngine_PropagatesRunnerError(t *testing.T) {
	runnerErr := errors.New("runner failed")
	engine, err := NewReplayEngine(&recordingRunner{err: runnerErr})
	if err != nil {
		t.Fatalf("NewReplayEngine() error = %v", err)
	}

	_, err = engine.Replay(context.Background(), engineTestScenario())
	if !errors.Is(err, runnerErr) {
		t.Fatalf("Replay() error = %v, want %v", err, runnerErr)
	}
}

func TestReplayEngine_RepeatedRunsAreDeterministicAndIsolated(t *testing.T) {
	runner := testRunner(t)
	engine, err := NewReplayEngine(runner)
	if err != nil {
		t.Fatalf("NewReplayEngine() error = %v", err)
	}
	scenario := engineTestScenario()

	first, err := engine.Replay(context.Background(), scenario)
	if err != nil {
		t.Fatalf("first Replay() error = %v", err)
	}
	second, err := engine.Replay(context.Background(), scenario)
	if err != nil {
		t.Fatalf("second Replay() error = %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated replay results differ: first = %+v, second = %+v", first, second)
	}
}

type recordingRunner struct {
	called   bool
	scenario replaydomain.Scenario
	result   Result
	err      error
}

func (r *recordingRunner) Run(
	_ context.Context,
	scenario replaydomain.Scenario,
) (Result, error) {
	r.called = true
	r.scenario = scenario
	return r.result, r.err
}

func engineTestScenario() replaydomain.Scenario {
	return replaydomain.Scenario{
		ID:                 "scenario-1",
		Provider:           replaydomain.ProviderConfiguration{ID: "fake"},
		InitialVirtualTime: time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC),
	}
}

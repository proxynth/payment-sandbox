package application

import (
	"context"

	replaydomain "proxynth/payment-sandbox/internal/replay/domain"
)

// ScenarioRunner executes one validated scenario and returns its observable
// execution state.
type ScenarioRunner interface {
	Run(context.Context, replaydomain.Scenario) (Result, error)
}

// ReplayEngine is the application boundary for deterministic scenario replay.
// It owns replay lifecycle validation and delegates business execution to the
// scenario runner.
type ReplayEngine struct {
	runner ScenarioRunner
}

func NewReplayEngine(runner ScenarioRunner) (*ReplayEngine, error) {
	if runner == nil {
		return nil, ErrNilScenarioRunner
	}

	return &ReplayEngine{runner: runner}, nil
}

func (engine *ReplayEngine) Replay(
	ctx context.Context,
	scenario replaydomain.Scenario,
) (Result, error) {
	if err := scenario.Validate(); err != nil {
		return Result{}, err
	}

	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	return engine.runner.Run(ctx, scenario)
}

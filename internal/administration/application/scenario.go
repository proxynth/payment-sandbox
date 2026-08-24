package application

import (
	"context"
	"errors"

	replaydomain "proxynth/payment-sandbox/internal/replay/domain"
)

var ErrScenarioNotFound = errors.New("scenario not found")

type ScenarioRepository interface {
	FindByID(context.Context, replaydomain.ScenarioID) (*replaydomain.Scenario, error)
}

type ScenarioInspection struct {
	repository ScenarioRepository
}

func NewScenarioInspection(repository ScenarioRepository) (*ScenarioInspection, error) {
	if repository == nil {
		return nil, errors.New("scenario repository is nil")
	}

	return &ScenarioInspection{repository: repository}, nil
}

func (s *ScenarioInspection) Execute(
	ctx context.Context,
	id replaydomain.ScenarioID,
) (*replaydomain.Scenario, error) {
	scenario, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if scenario == nil {
		return nil, ErrScenarioNotFound
	}
	if err := scenario.Validate(); err != nil {
		return nil, err
	}

	return scenario, nil
}

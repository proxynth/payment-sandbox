package application

import (
	"context"
	"errors"

	"proxynth/payment-sandbox/internal/replay/domain"
)

type ScenarioRepository interface {
	Save(context.Context, *domain.Scenario) error
	FindByID(context.Context, domain.ScenarioID) (*domain.Scenario, error)
}

type ScenarioService struct {
	repository ScenarioRepository
	engine     *ReplayEngine
}

func NewScenarioService(repository ScenarioRepository, engine *ReplayEngine) (*ScenarioService, error) {
	if repository == nil || engine == nil {
		return nil, errors.New("invalid scenario service")
	}
	return &ScenarioService{repository: repository, engine: engine}, nil
}

func (s *ScenarioService) Create(ctx context.Context, scenario *domain.Scenario) error {
	if scenario == nil {
		return domain.ErrInvalidScenarioID
	}
	if err := scenario.Validate(); err != nil {
		return err
	}
	if err := s.repository.Save(ctx, scenario); err != nil {
		return err
	}
	return nil
}

func (s *ScenarioService) Execute(ctx context.Context, id domain.ScenarioID) (Result, error) {
	scenario, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return Result{}, err
	}
	if scenario == nil {
		return Result{}, ErrScenarioNotFound
	}
	return s.engine.Replay(ctx, *scenario)
}

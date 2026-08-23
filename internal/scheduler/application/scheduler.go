package application

import (
	"context"
	"errors"
	"time"

	businessclock "proxynth/payment-sandbox/internal/platform/clock"
	"proxynth/payment-sandbox/internal/scheduler/domain"
)

type Repository interface {
	FindExecutable(ctx context.Context, at time.Time, limit int) ([]*domain.Job, error)
	Acquire(ctx context.Context, id domain.JobID, owner string, expiresAt time.Time) (*domain.Job, error)
}

type Dispatcher interface {
	Dispatch(ctx context.Context, job *domain.Job) error
}

type Config struct {
	Owner         string
	BatchSize     int
	LeaseDuration time.Duration
}

type Scheduler struct {
	repository    Repository
	dispatcher    Dispatcher
	clock         businessclock.Clock
	owner         string
	batchSize     int
	leaseDuration time.Duration
}

func NewScheduler(
	repository Repository,
	dispatcher Dispatcher,
	clock businessclock.Clock,
	config Config,
) (*Scheduler, error) {
	if repository == nil {
		return nil, ErrInvalidRepository
	}

	if dispatcher == nil {
		return nil, ErrInvalidDispatcher
	}

	if clock == nil {
		return nil, ErrInvalidClock
	}

	if config.Owner == "" {
		return nil, ErrInvalidOwner
	}

	if config.BatchSize <= 0 {
		return nil, ErrInvalidBatchSize
	}

	if config.LeaseDuration <= 0 {
		return nil, ErrInvalidLeaseDuration
	}

	return &Scheduler{
		repository:    repository,
		dispatcher:    dispatcher,
		clock:         clock,
		owner:         config.Owner,
		batchSize:     config.BatchSize,
		leaseDuration: config.LeaseDuration,
	}, nil
}

func (s *Scheduler) Tick(ctx context.Context) error {
	now := s.clock.Now().UTC()
	jobs, err := s.repository.FindExecutable(ctx, now, s.batchSize)
	if err != nil {
		return err
	}

	var tickErrors []error
	for _, job := range jobs {
		if job == nil {
			tickErrors = append(tickErrors, ErrNilAcquiredJob)
			continue
		}

		leaseExpiresAt := now.Add(s.leaseDuration)
		acquired, err := s.repository.Acquire(ctx, job.ID(), s.owner, leaseExpiresAt)
		if err != nil {
			tickErrors = append(tickErrors, err)
			continue
		}

		if acquired == nil {
			tickErrors = append(tickErrors, ErrNilAcquiredJob)
			continue
		}

		if err := s.dispatcher.Dispatch(ctx, acquired); err != nil {
			tickErrors = append(tickErrors, err)
		}
	}

	return errors.Join(tickErrors...)
}

package application

import (
	"context"
	"errors"

	businessclock "proxynth/payment-sandbox/internal/platform/clock"
	"proxynth/payment-sandbox/internal/scheduler/domain"
)

type JobHandler func(ctx context.Context, payload []byte) error

type WorkerRepository interface {
	Save(ctx context.Context, job *domain.Job) error
}

type Worker struct {
	repository  WorkerRepository
	handlers    map[domain.JobType]JobHandler
	retryPolicy domain.RetryPolicy
	clock       businessclock.Clock
}

func NewWorker(
	repository WorkerRepository,
	handlers map[domain.JobType]JobHandler,
) (*Worker, error) {
	if repository == nil {
		return nil, ErrInvalidWorkerRepository
	}

	workerHandlers := make(map[domain.JobType]JobHandler, len(handlers))
	for jobType, handler := range handlers {
		if handler == nil {
			return nil, ErrInvalidHandler
		}

		workerHandlers[jobType] = handler
	}

	return &Worker{
		repository: repository,
		handlers:   workerHandlers,
	}, nil
}

func NewWorkerWithRetry(repository WorkerRepository, handlers map[domain.JobType]JobHandler, retryPolicy domain.RetryPolicy, businessClock businessclock.Clock) (*Worker, error) {
	worker, err := NewWorker(repository, handlers)
	if err != nil {
		return nil, err
	}
	if retryPolicy == nil || businessClock == nil {
		return nil, errors.New("invalid scheduler retry configuration")
	}
	worker.retryPolicy = retryPolicy
	worker.clock = businessClock
	return worker, nil
}

func (w *Worker) Execute(ctx context.Context, job *domain.Job) error {
	if job == nil {
		return ErrNilJob
	}

	handler, ok := w.handlers[job.Type()]
	if !ok {
		return ErrUnknownJobType
	}

	if err := job.Start(); err != nil {
		return err
	}

	if err := w.repository.Save(ctx, job); err != nil {
		return err
	}

	if err := handler(ctx, job.Payload()); err != nil {
		return w.persistFailure(ctx, job, err)
	}

	if err := job.Complete(); err != nil {
		return err
	}

	return w.repository.Save(ctx, job)
}

func (w *Worker) persistFailure(ctx context.Context, job *domain.Job, executionErr error) error {
	if err := job.Fail(); err != nil {
		return errors.Join(executionErr, err)
	}

	if w.retryPolicy != nil && w.clock != nil {
		decision, err := w.retryPolicy.Decide(job.Attempts())
		if err != nil {
			return errors.Join(executionErr, err)
		}
		if decision.Retry {
			if err := job.ScheduleRetry(w.clock.Now().Add(decision.RetryAfter)); err != nil {
				return errors.Join(executionErr, err)
			}
		}
	}
	return errors.Join(executionErr, w.repository.Save(ctx, job))
}

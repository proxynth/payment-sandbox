package application

import (
	"context"
	"errors"

	"proxynth/payment-sandbox/internal/scheduler/domain"
)

type JobHandler func(ctx context.Context, payload []byte) error

type WorkerRepository interface {
	Save(ctx context.Context, job *domain.Job) error
}

type Worker struct {
	repository WorkerRepository
	handlers   map[domain.JobType]JobHandler
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

	return errors.Join(executionErr, w.repository.Save(ctx, job))
}

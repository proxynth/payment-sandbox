package sqlite

import (
	"context"
	"encoding/json"
	"fmt"

	"proxynth/payment-sandbox/internal/paymentworkflow/domain"
	schedulerdomain "proxynth/payment-sandbox/internal/scheduler/domain"
)

type JobStore interface {
	Save(context.Context, *schedulerdomain.Job) error
}

type Publisher struct{ jobs JobStore }

func NewPublisher(jobs JobStore) *Publisher { return &Publisher{jobs: jobs} }

func (p *Publisher) Publish(ctx context.Context, message domain.Message) error {
	if err := message.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal saga message: %w", err)
	}
	job, err := schedulerdomain.NewJob(schedulerdomain.JobID(message.ID), "saga.step", payload, message.ScheduledAt)
	if err != nil {
		return err
	}
	return p.jobs.Save(ctx, &job)
}

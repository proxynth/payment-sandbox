package application

import (
	"context"

	"proxynth/payment-sandbox/internal/payment/domain"
)

var ErrEventAlreadyExists = &eventLogError{"event already exists"}

type EventLog interface {
	Append(context.Context, domain.BusinessEvent) error
	ListByAggregate(context.Context, domain.ID) ([]domain.BusinessEvent, error)
}

// EventPublisher receives completed payment transitions and may project them
// into the event log and durable asynchronous work.
type EventPublisher interface {
	Publish(context.Context, *domain.Payment, domain.EventType) error
}

type eventLogError struct{ message string }

func (e *eventLogError) Error() string { return e.message }

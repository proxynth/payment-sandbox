package domain

import "time"

type EventID string

type EventType string

const (
	EventPaymentCreated    EventType = "payment.created"
	EventPaymentAuthorized EventType = "payment.authorized"
	EventPaymentFailed     EventType = "payment.failed"
	EventPaymentCancelled  EventType = "payment.cancelled"
	EventPaymentCaptured   EventType = "payment.captured"
	EventPaymentRefunded   EventType = "payment.refunded"
)

type BusinessEvent struct {
	id               EventID
	aggregateID      ID
	eventType        EventType
	occurredAt       time.Time
	aggregateVersion uint64
	correlationID    string
	causationID      EventID
}

func NewBusinessEvent(
	id EventID,
	aggregateID ID,
	eventType EventType,
	occurredAt time.Time,
	aggregateVersion uint64,
	correlationID string,
	causationID EventID,
) (BusinessEvent, error) {
	if id == "" {
		return BusinessEvent{}, ErrInvalidEventID
	}

	if aggregateID == "" {
		return BusinessEvent{}, ErrInvalidAggregateID
	}

	if !eventType.Valid() {
		return BusinessEvent{}, ErrInvalidEventType
	}

	if occurredAt.IsZero() {
		return BusinessEvent{}, ErrInvalidEventTimestamp
	}

	if aggregateVersion == 0 {
		return BusinessEvent{}, ErrInvalidAggregateVersion
	}

	return BusinessEvent{
		id:               id,
		aggregateID:      aggregateID,
		eventType:        eventType,
		occurredAt:       occurredAt.UTC(),
		aggregateVersion: aggregateVersion,
		correlationID:    correlationID,
		causationID:      causationID,
	}, nil
}

func (t EventType) Valid() bool {
	switch t {
	case EventPaymentCreated,
		EventPaymentAuthorized,
		EventPaymentFailed,
		EventPaymentCancelled,
		EventPaymentCaptured,
		EventPaymentRefunded:
		return true
	default:
		return false
	}
}

func (e BusinessEvent) ID() EventID {
	return e.id
}

func (e BusinessEvent) AggregateID() ID {
	return e.aggregateID
}

func (e BusinessEvent) Type() EventType {
	return e.eventType
}

func (e BusinessEvent) OccurredAt() time.Time {
	return e.occurredAt
}

func (e BusinessEvent) AggregateVersion() uint64 {
	return e.aggregateVersion
}

func (e BusinessEvent) CorrelationID() string {
	return e.correlationID
}

func (e BusinessEvent) CausationID() EventID {
	return e.causationID
}

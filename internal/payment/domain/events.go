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
	event := BusinessEvent{
		id:               id,
		aggregateID:      aggregateID,
		eventType:        eventType,
		occurredAt:       occurredAt.UTC(),
		aggregateVersion: aggregateVersion,
		correlationID:    correlationID,
		causationID:      causationID,
	}
	if err := event.Validate(); err != nil {
		return BusinessEvent{}, err
	}

	return event, nil
}

func (e BusinessEvent) Validate() error {
	if e.id == "" {
		return ErrInvalidEventID
	}

	if e.aggregateID == "" {
		return ErrInvalidAggregateID
	}

	if !e.eventType.Valid() {
		return ErrInvalidEventType
	}

	if e.occurredAt.IsZero() {
		return ErrInvalidEventTimestamp
	}

	if e.aggregateVersion == 0 {
		return ErrInvalidAggregateVersion
	}

	return nil
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

package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/payment/application"
	"proxynth/payment-sandbox/internal/payment/domain"
)

func TestEventLogRepository_AppendAndListByAggregate(t *testing.T) {
	repository := NewEventLogRepository(newTestDatabase(t))
	aggregateID := domain.ID("payment-event-log")

	first := newEvent(t, "event-1", aggregateID, domain.EventPaymentCreated, time.Unix(10, 0), 1)
	second := newEvent(t, "event-2", aggregateID, domain.EventPaymentAuthorized, time.Unix(20, 0), 2)
	if err := repository.Append(context.Background(), first); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	if err := repository.Append(context.Background(), second); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}

	events, err := repository.ListByAggregate(context.Background(), aggregateID)
	if err != nil {
		t.Fatalf("ListByAggregate() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].ID() != first.ID() || events[1].ID() != second.ID() {
		t.Fatalf("event order = [%s, %s], want [%s, %s]", events[0].ID(), events[1].ID(), first.ID(), second.ID())
	}
	if events[1].CorrelationID() != second.CorrelationID() || events[1].CausationID() != second.CausationID() {
		t.Fatal("event context metadata was not preserved")
	}
}

func TestEventLogRepository_OrdersEqualTimestampsDeterministically(t *testing.T) {
	repository := NewEventLogRepository(newTestDatabase(t))
	aggregateID := domain.ID("payment-event-order")
	at := time.Unix(10, 0)

	laterVersion := newEvent(t, "event-2", aggregateID, domain.EventPaymentAuthorized, at, 2)
	earlierVersion := newEvent(t, "event-1", aggregateID, domain.EventPaymentCreated, at, 1)
	if err := repository.Append(context.Background(), laterVersion); err != nil {
		t.Fatalf("Append(laterVersion) error = %v", err)
	}
	if err := repository.Append(context.Background(), earlierVersion); err != nil {
		t.Fatalf("Append(earlierVersion) error = %v", err)
	}

	events, err := repository.ListByAggregate(context.Background(), aggregateID)
	if err != nil {
		t.Fatalf("ListByAggregate() error = %v", err)
	}
	if events[0].AggregateVersion() != 1 || events[1].AggregateVersion() != 2 {
		t.Fatalf("versions = [%d, %d], want [1, 2]", events[0].AggregateVersion(), events[1].AggregateVersion())
	}
}

func TestEventLogRepository_RejectsDuplicateEventID(t *testing.T) {
	repository := NewEventLogRepository(newTestDatabase(t))
	event := newEvent(t, "event-duplicate", "payment-duplicate", domain.EventPaymentCreated, time.Unix(1, 0), 1)
	if err := repository.Append(context.Background(), event); err != nil {
		t.Fatalf("first Append() error = %v", err)
	}

	err := repository.Append(context.Background(), event)
	if !errors.Is(err, application.ErrEventAlreadyExists) {
		t.Fatalf("second Append() error = %v, want %v", err, application.ErrEventAlreadyExists)
	}
}

func TestEventLogRepository_RejectsInvalidEventBeforePersistence(t *testing.T) {
	repository := NewEventLogRepository(newTestDatabase(t))

	if err := repository.Append(context.Background(), domain.BusinessEvent{}); !errors.Is(err, domain.ErrInvalidEventID) {
		t.Fatalf("Append() error = %v, want %v", err, domain.ErrInvalidEventID)
	}
}

func TestEventLogRepository_ReturnsEmptyHistoryForUnknownAggregate(t *testing.T) {
	repository := NewEventLogRepository(newTestDatabase(t))

	events, err := repository.ListByAggregate(context.Background(), "payment-missing")
	if err != nil {
		t.Fatalf("ListByAggregate() error = %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("len(events) = %d, want 0", len(events))
	}
}

func TestEventLogRepository_PropagatesCanceledContext(t *testing.T) {
	repository := NewEventLogRepository(newTestDatabase(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	event := newEvent(t, "event-canceled", "payment-canceled", domain.EventPaymentCreated, time.Unix(1, 0), 1)
	if err := repository.Append(ctx, event); err == nil {
		t.Fatal("Append() error = nil, want canceled context error")
	}
}

func newEvent(
	t *testing.T,
	id domain.EventID,
	aggregateID domain.ID,
	eventType domain.EventType,
	at time.Time,
	version uint64,
) domain.BusinessEvent {
	t.Helper()
	event, err := domain.NewBusinessEvent(id, aggregateID, eventType, at, version, "correlation-1", "causation-1")
	if err != nil {
		t.Fatalf("NewBusinessEvent() error = %v", err)
	}
	return event
}

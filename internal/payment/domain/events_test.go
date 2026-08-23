package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewBusinessEvent(t *testing.T) {
	occurredAt := time.Date(2026, time.July, 29, 12, 34, 56, 123, time.FixedZone("CEST", 2*60*60))
	event, err := NewBusinessEvent(
		"event-1",
		"payment-1",
		EventPaymentAuthorized,
		occurredAt,
		2,
		"request-1",
		"event-0",
	)
	if err != nil {
		t.Fatalf("NewBusinessEvent() error = %v", err)
	}

	if event.ID() != "event-1" {
		t.Errorf("ID() = %q, want %q", event.ID(), "event-1")
	}

	if event.AggregateID() != "payment-1" {
		t.Errorf("AggregateID() = %q, want %q", event.AggregateID(), "payment-1")
	}

	if event.Type() != EventPaymentAuthorized {
		t.Errorf("Type() = %q, want %q", event.Type(), EventPaymentAuthorized)
	}

	if !event.OccurredAt().Equal(occurredAt) || event.OccurredAt().Location() != time.UTC {
		t.Errorf("OccurredAt() = %v, want the UTC equivalent of %v", event.OccurredAt(), occurredAt)
	}

	if event.AggregateVersion() != 2 {
		t.Errorf("AggregateVersion() = %d, want %d", event.AggregateVersion(), 2)
	}

	if event.CorrelationID() != "request-1" {
		t.Errorf("CorrelationID() = %q, want %q", event.CorrelationID(), "request-1")
	}

	if event.CausationID() != "event-0" {
		t.Errorf("CausationID() = %q, want %q", event.CausationID(), "event-0")
	}
}

func TestNewBusinessEvent_AllPaymentLifecycleTypesAreValid(t *testing.T) {
	types := []EventType{
		EventPaymentCreated,
		EventPaymentAuthorized,
		EventPaymentFailed,
		EventPaymentCancelled,
		EventPaymentCaptured,
		EventPaymentRefunded,
	}

	for _, eventType := range types {
		if !eventType.Valid() {
			t.Errorf("EventType.Valid() = false for %q", eventType)
		}
	}
}

func TestNewBusinessEvent_RejectsInvalidMetadata(t *testing.T) {
	validTime := time.Unix(1, 0)
	tests := []struct {
		name      string
		id        EventID
		agg       ID
		eventType EventType
		at        time.Time
		ver       uint64
		want      error
	}{
		{name: "empty event id", agg: "payment-1", eventType: EventPaymentCreated, at: validTime, ver: 1, want: ErrInvalidEventID},
		{name: "empty aggregate id", id: "event-1", eventType: EventPaymentCreated, at: validTime, ver: 1, want: ErrInvalidAggregateID},
		{name: "unknown event type", id: "event-1", agg: "payment-1", eventType: "payment.unknown", at: validTime, ver: 1, want: ErrInvalidEventType},
		{name: "zero timestamp", id: "event-1", agg: "payment-1", eventType: EventPaymentCreated, at: time.Time{}, ver: 1, want: ErrInvalidEventTimestamp},
		{name: "zero aggregate version", id: "event-1", agg: "payment-1", eventType: EventPaymentCreated, at: validTime, want: ErrInvalidAggregateVersion},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewBusinessEvent(tt.id, tt.agg, tt.eventType, tt.at, tt.ver, "", "")
			if !errors.Is(err, tt.want) {
				t.Fatalf("NewBusinessEvent() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestBusinessEvent_IsImmutableFromCallers(t *testing.T) {
	event, err := NewBusinessEvent(
		"event-1",
		"payment-1",
		EventPaymentCreated,
		time.Unix(1, 0),
		1,
		"request-1",
		"",
	)
	if err != nil {
		t.Fatalf("NewBusinessEvent() error = %v", err)
	}

	copy := event
	if copy.ID() != event.ID() || copy.AggregateVersion() != event.AggregateVersion() {
		t.Fatal("copy of BusinessEvent changed its original value")
	}
}

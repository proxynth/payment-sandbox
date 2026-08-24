package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/payment/domain"
)

func TestTimeline_CombinesCurrentPaymentAndEvents(t *testing.T) {
	payment, err := domain.New("payment-timeline", mustMoney(t))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	event, err := domain.NewBusinessEvent(
		"event-timeline",
		payment.ID(),
		domain.EventPaymentCreated,
		time.Unix(1, 0),
		1,
		"correlation-1",
		"",
	)
	if err != nil {
		t.Fatalf("NewBusinessEvent() error = %v", err)
	}

	payments := &timelinePaymentRepository{payment: payment}
	events := &timelineEventLog{events: []domain.BusinessEvent{event}}
	timeline, err := NewTimeline(payments, events)
	if err != nil {
		t.Fatalf("NewTimeline() error = %v", err)
	}

	result, err := timeline.Execute(context.Background(), payment.ID())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Payment != payment || len(result.Events) != 1 || result.Events[0].ID() != event.ID() {
		t.Fatalf("result = %#v", result)
	}
}

func TestNewTimeline_RejectsNilDependencies(t *testing.T) {
	events := &timelineEventLog{}
	payments := &timelinePaymentRepository{}
	if _, err := NewTimeline(nil, events); !errors.Is(err, ErrNilPaymentRepository) {
		t.Fatalf("nil payments error = %v", err)
	}
	if _, err := NewTimeline(payments, nil); !errors.Is(err, ErrNilEventLog) {
		t.Fatalf("nil event log error = %v", err)
	}
}

func mustMoney(t *testing.T) domain.Money {
	t.Helper()
	money, err := domain.NewMoney(1000, "EUR")
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}
	return money
}

type timelinePaymentRepository struct {
	payment *domain.Payment
}

func (r *timelinePaymentRepository) Save(_ context.Context, _ *domain.Payment) error { return nil }

func (r *timelinePaymentRepository) FindByID(_ context.Context, _ domain.ID) (*domain.Payment, error) {
	if r.payment == nil {
		return nil, errors.New("payment not found")
	}
	return r.payment, nil
}

type timelineEventLog struct {
	events []domain.BusinessEvent
}

func (l *timelineEventLog) Append(_ context.Context, _ domain.BusinessEvent) error { return nil }

func (l *timelineEventLog) ListByAggregate(_ context.Context, _ domain.ID) ([]domain.BusinessEvent, error) {
	return l.events, nil
}

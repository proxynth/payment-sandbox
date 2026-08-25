package app

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	"proxynth/payment-sandbox/internal/platform/clock"
	schedulerdomain "proxynth/payment-sandbox/internal/scheduler/domain"
	webhookmemory "proxynth/payment-sandbox/internal/webhook/adapters/memory"
	webhookapplication "proxynth/payment-sandbox/internal/webhook/application"
	webhookdomain "proxynth/payment-sandbox/internal/webhook/domain"
)

type paymentEventLogFake struct {
	events []paymentdomain.BusinessEvent
}

func (f *paymentEventLogFake) Append(_ context.Context, event paymentdomain.BusinessEvent) error {
	f.events = append(f.events, event)
	return nil
}

func (f *paymentEventLogFake) ListByAggregate(context.Context, paymentdomain.ID) ([]paymentdomain.BusinessEvent, error) {
	return f.events, nil
}

type paymentEventJobsFake struct {
	jobs []*schedulerdomain.Job
}

func (f *paymentEventJobsFake) Save(_ context.Context, job *schedulerdomain.Job) error {
	f.jobs = append(f.jobs, job)
	return nil
}

func TestPaymentEventPublisherRecordsEventAndSchedulesWebhook(t *testing.T) {
	when := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	virtualClock, err := clock.NewVirtualClock(when)
	if err != nil {
		t.Fatalf("NewVirtualClock() error = %v", err)
	}
	endpoints := webhookmemory.NewRepository()
	endpoint, err := webhookdomain.NewEndpoint("endpoint-1", "https://example.test/hooks")
	if err != nil {
		t.Fatalf("NewEndpoint() error = %v", err)
	}
	if err := endpoints.Save(context.Background(), endpoint); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	events := &paymentEventLogFake{}
	jobs := &paymentEventJobsFake{}
	publisher, err := newPaymentEventPublisher(events, endpoints, jobs, virtualClock)
	if err != nil {
		t.Fatalf("newPaymentEventPublisher() error = %v", err)
	}
	money, err := paymentdomain.NewMoney(1000, "EUR")
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}
	payment, err := paymentdomain.New("payment-1", money)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := publisher.Publish(context.Background(), payment, paymentdomain.EventPaymentCreated); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if len(events.events) != 1 || events.events[0].Type() != paymentdomain.EventPaymentCreated {
		t.Fatalf("events = %+v, want one payment.created event", events.events)
	}
	if len(jobs.jobs) != 1 || jobs.jobs[0].Type() != webhookapplication.DeliveryJobType {
		t.Fatalf("jobs = %+v, want one webhook delivery job", jobs.jobs)
	}
	var payload webhookapplication.DeliveryPayload
	if err := json.Unmarshal(jobs.jobs[0].Payload(), &payload); err != nil {
		t.Fatalf("decode job payload: %v", err)
	}
	if payload.EndpointID != endpoint.ID() {
		t.Fatalf("endpoint id = %q, want %q", payload.EndpointID, endpoint.ID())
	}
	var body paymentWebhookEvent
	if err := json.Unmarshal(payload.Body, &body); err != nil {
		t.Fatalf("decode webhook body: %v", err)
	}
	if body.PaymentID != payment.ID() || body.Type != paymentdomain.EventPaymentCreated || !body.OccurredAt.Equal(when) {
		t.Fatalf("webhook body = %+v", body)
	}
}

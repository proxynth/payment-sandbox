package application

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	schedulerdomain "proxynth/payment-sandbox/internal/scheduler/domain"
	webhookapplication "proxynth/payment-sandbox/internal/webhook/application"
)

// Invariant: the read-only runtime view joins only durable causal records and
// never needs to execute a job or call an external endpoint.
func TestRuntimeHistoryReconstructsPaymentAndCausalJobs(t *testing.T) {
	money, _ := paymentdomain.NewMoney(1000, "EUR")
	payment, _ := paymentdomain.New("payment-history", money)
	payload, _ := json.Marshal(paymentdomain.NewEventSnapshot(payment))
	event, _ := paymentdomain.NewBusinessEventWithPayload("event-history", payment.ID(), paymentdomain.EventPaymentCreated, time.Unix(1, 0), 1, "corr", "", payload)
	when := time.Unix(1, 0).UTC()
	snapshot := schedulerdomain.JobSnapshot{ID: "job-history", Type: "webhook.delivery", Payload: []byte("sensitive"), ScheduledAt: when, NextAttemptAt: when, Status: schedulerdomain.JobCompleted, Attempts: 1, AggregateID: string(payment.ID()), CausationID: string(event.ID())}
	history, err := NewRuntimeHistory(historyTestTransaction{}, &runtimeHistoryEvents{events: []paymentdomain.BusinessEvent{event}}, &runtimeHistoryJobs{snapshots: []schedulerdomain.JobSnapshot{snapshot}}, &runtimeHistoryHooks{attempts: []webhookapplication.DeliveryAttempt{{JobID: "job-history", Attempt: 1, Outcome: webhookapplication.DeliverySucceeded}}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := history.Execute(context.Background(), payment.ID())
	if err != nil {
		t.Fatal(err)
	}
	if result.Payment.ID() != payment.ID() || len(result.Events) != 1 || len(result.Jobs) != 1 || len(result.Jobs[0].Deliveries) != 1 {
		t.Fatalf("result=%#v", result)
	}
}

type runtimeHistoryEvents struct{ events []paymentdomain.BusinessEvent }

type historyTestTransaction struct{}

func (historyTestTransaction) WithinContext(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (r *runtimeHistoryEvents) Append(context.Context, paymentdomain.BusinessEvent) error { return nil }
func (r *runtimeHistoryEvents) ListByAggregate(context.Context, paymentdomain.ID) ([]paymentdomain.BusinessEvent, error) {
	return r.events, nil
}

type runtimeHistoryJobs struct{ snapshots []schedulerdomain.JobSnapshot }

func (r *runtimeHistoryJobs) ListAuditByAggregate(context.Context, string) ([]schedulerdomain.JobSnapshot, error) {
	return r.snapshots, nil
}

type runtimeHistoryHooks struct {
	attempts []webhookapplication.DeliveryAttempt
}

func (r *runtimeHistoryHooks) ListByJob(context.Context, string) ([]webhookapplication.DeliveryAttempt, error) {
	return r.attempts, nil
}

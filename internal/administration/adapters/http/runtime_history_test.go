package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/api"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	schedulerdomain "proxynth/payment-sandbox/internal/scheduler/domain"
	webhookapplication "proxynth/payment-sandbox/internal/webhook/application"
)

// Invariant: the administrative runtime history is observable and excludes
// durable job payloads and callback bodies.
func TestRuntimeHistoryHandlerReturnsSafeReadOnlyHistory(t *testing.T) {
	money, _ := paymentdomain.NewMoney(1000, "EUR")
	payment, _ := paymentdomain.New("payment-http-history", money)
	payload, _ := json.Marshal(paymentdomain.NewEventSnapshot(payment))
	event, _ := paymentdomain.NewBusinessEventWithPayload("event-http-history", payment.ID(), paymentdomain.EventPaymentCreated, time.Unix(1, 0), 1, "corr", "", payload)
	handler, err := NewRuntimeHistoryHandler(&httpHistoryEvents{events: []paymentdomain.BusinessEvent{event}}, &httpHistoryJobs{snapshots: []schedulerdomain.JobSnapshot{{ID: "job-http", Type: "webhook.delivery", Payload: []byte("secret-payload"), ScheduledAt: time.Unix(1, 0), NextAttemptAt: time.Unix(1, 0), Status: schedulerdomain.JobCompleted, AggregateID: string(payment.ID()), CausationID: string(event.ID())}}}, &httpHistoryHooks{attempts: []webhookapplication.DeliveryAttempt{{JobID: "job-http", Attempt: 1, Outcome: webhookapplication.DeliverySucceeded}}})
	if err != nil {
		t.Fatal(err)
	}
	server, _ := api.NewServer(":8080")
	if err := handler.Register(server, "test-admin-token"); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	server.ServeHTTP(response, adminRequest(http.MethodGet, "/admin/runtime-history/payments/"+string(payment.ID())))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, expected := range []string{`"id":"payment-http-history"`, `"event-http-history"`, `"job-http"`, `"succeeded"`} {
		if !strings.Contains(body, expected) {
			t.Errorf("body=%q missing %q", body, expected)
		}
	}
	if strings.Contains(body, "secret-payload") || strings.Contains(body, "payload") {
		t.Fatalf("response exposed job payload: %s", body)
	}
}

type httpHistoryEvents struct{ events []paymentdomain.BusinessEvent }

func (r *httpHistoryEvents) Append(context.Context, paymentdomain.BusinessEvent) error { return nil }
func (r *httpHistoryEvents) ListByAggregate(context.Context, paymentdomain.ID) ([]paymentdomain.BusinessEvent, error) {
	return r.events, nil
}

type httpHistoryJobs struct{ snapshots []schedulerdomain.JobSnapshot }

func (r *httpHistoryJobs) ListAuditByAggregate(context.Context, string) ([]schedulerdomain.JobSnapshot, error) {
	return r.snapshots, nil
}

type httpHistoryHooks struct {
	attempts []webhookapplication.DeliveryAttempt
}

func (r *httpHistoryHooks) ListByJob(context.Context, string) ([]webhookapplication.DeliveryAttempt, error) {
	return r.attempts, nil
}

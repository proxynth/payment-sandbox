package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/api"
	paymentapplication "proxynth/payment-sandbox/internal/payment/application"
	"proxynth/payment-sandbox/internal/payment/domain"
)

func TestTimelineHandler_ReturnsPaymentTimeline(t *testing.T) {
	payment, err := domain.New("payment-timeline", timelineMoney(t))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	event, err := domain.NewBusinessEvent(
		"event-timeline",
		payment.ID(),
		domain.EventPaymentCreated,
		time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		1,
		"correlation-1",
		"",
	)
	if err != nil {
		t.Fatalf("NewBusinessEvent() error = %v", err)
	}

	handler, err := NewTimelineHandler(
		&httpTimelinePaymentRepository{payment: payment},
		&httpTimelineEventLog{events: []domain.BusinessEvent{event}},
	)
	if err != nil {
		t.Fatalf("NewTimelineHandler() error = %v", err)
	}
	server, err := api.NewServer(":8080")
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if err := handler.Register(server, "test-admin-token"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, adminRequest(http.MethodGet, "/admin/payments/payment-timeline/timeline"))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	for _, expected := range []string{`"payment"`, `"id":"payment-timeline"`, `"events"`, `"id":"event-timeline"`, `"type":"payment.created"`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("body = %q, missing %q", response.Body.String(), expected)
		}
	}
}

func TestTimelineHandler_RejectsMalformedPath(t *testing.T) {
	payment, err := domain.New("payment-timeline", timelineMoney(t))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	handler, err := NewTimelineHandler(&httpTimelinePaymentRepository{payment: payment}, &httpTimelineEventLog{})
	if err != nil {
		t.Fatalf("NewTimelineHandler() error = %v", err)
	}
	server, err := api.NewServer(":8080")
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if err := handler.Register(server, "test-admin-token"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, adminRequest(http.MethodGet, "/admin/payments/payment-timeline"))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestTimelineHandler_MapsMissingPayment(t *testing.T) {
	handler, err := NewTimelineHandler(&httpTimelinePaymentRepository{}, &httpTimelineEventLog{})
	if err != nil {
		t.Fatalf("NewTimelineHandler() error = %v", err)
	}
	server, err := api.NewServer(":8080")
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if err := handler.Register(server, "test-admin-token"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, adminRequest(http.MethodGet, "/admin/payments/payment-missing/timeline"))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func timelineMoney(t *testing.T) domain.Money {
	t.Helper()
	money, err := domain.NewMoney(1000, "EUR")
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}
	return money
}

type httpTimelinePaymentRepository struct {
	payment *domain.Payment
}

func (r *httpTimelinePaymentRepository) Save(_ context.Context, _ *domain.Payment) error { return nil }

func (r *httpTimelinePaymentRepository) FindByID(_ context.Context, _ domain.ID) (*domain.Payment, error) {
	if r.payment == nil {
		return nil, paymentapplication.ErrPaymentNotFound
	}
	return r.payment, nil
}

type httpTimelineEventLog struct {
	events []domain.BusinessEvent
}

func (l *httpTimelineEventLog) Append(_ context.Context, _ domain.BusinessEvent) error { return nil }

func (l *httpTimelineEventLog) ListByAggregate(_ context.Context, _ domain.ID) ([]domain.BusinessEvent, error) {
	return l.events, nil
}

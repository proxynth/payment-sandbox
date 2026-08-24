package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/api"
	"proxynth/payment-sandbox/internal/payment/domain"
	"proxynth/payment-sandbox/internal/platform/clock"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
)

func TestDiagnosticsHandler_ReturnsStructuredSnapshot(t *testing.T) {
	payment, err := domain.New("payment-diagnostics", timelineMoney(t))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	virtualClock, err := clock.NewVirtualClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewVirtualClock() error = %v", err)
	}
	handler, err := NewDiagnosticsHandler(
		&httpTimelinePaymentRepository{payment: payment},
		&httpTimelineEventLog{},
		virtualClock,
		providerRegistryFake{ids: []providerdomain.ProviderID{"stripe", "adyen"}},
	)
	if err != nil {
		t.Fatalf("NewDiagnosticsHandler() error = %v", err)
	}
	server := diagnosticsServer(t, handler)

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/admin/diagnostics/payments/payment-diagnostics", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	for _, expected := range []string{`"payment"`, `"events"`, `"current_time":"2026-01-01T12:00:00Z"`, `"providers":["adyen","stripe"]`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("body = %q, missing %q", response.Body.String(), expected)
		}
	}
}

func TestDiagnosticsHandler_RejectsMalformedPath(t *testing.T) {
	virtualClock, err := clock.NewVirtualClock(time.Now())
	if err != nil {
		t.Fatalf("NewVirtualClock() error = %v", err)
	}
	handler, err := NewDiagnosticsHandler(&httpTimelinePaymentRepository{}, &httpTimelineEventLog{}, virtualClock, providerRegistryFake{})
	if err != nil {
		t.Fatalf("NewDiagnosticsHandler() error = %v", err)
	}
	server := diagnosticsServer(t, handler)

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/admin/diagnostics/payments/payment-diagnostics/extra", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func diagnosticsServer(t *testing.T, handler *DiagnosticsHandler) *api.Server {
	t.Helper()
	server, err := api.NewServer(":8080")
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if err := handler.Register(server); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	return server
}

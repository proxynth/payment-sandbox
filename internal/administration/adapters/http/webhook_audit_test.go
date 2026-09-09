package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"proxynth/payment-sandbox/internal/api"
	webhookapplication "proxynth/payment-sandbox/internal/webhook/application"
)

func TestWebhookAuditHandlerReturnsSafeAttemptDetails(t *testing.T) {
	handler, err := NewWebhookAuditHandler(&httpWebhookAuditReader{attempts: []webhookapplication.DeliveryAttempt{{
		JobID: "job-1", Attempt: 2, EndpointID: "endpoint-1", CorrelationID: "request-1", CausationID: "event-1",
		Outcome: webhookapplication.DeliveryFailed, HTTPStatus: http.StatusBadGateway, Error: "unexpected HTTP status 502",
	}}})
	if err != nil {
		t.Fatal(err)
	}
	server, err := api.NewServer(":8080")
	if err != nil {
		t.Fatal(err)
	}
	if err := handler.Register(server, "test-admin-token"); err != nil {
		t.Fatal(err)
	}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, adminRequest(http.MethodGet, "/admin/webhook-jobs/job-1/deliveries"))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	for _, expected := range []string{`"job_id":"job-1"`, `"attempt":2`, `"http_status":502`, `"error":"unexpected HTTP status 502"`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("body=%q missing %q", response.Body.String(), expected)
		}
	}
}

func TestWebhookAuditHandlerRejectsMalformedPath(t *testing.T) {
	handler, err := NewWebhookAuditHandler(&httpWebhookAuditReader{})
	if err != nil {
		t.Fatal(err)
	}
	server, _ := api.NewServer(":8080")
	if err := handler.Register(server, "test-admin-token"); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	server.ServeHTTP(response, adminRequest(http.MethodGet, "/admin/webhook-jobs/job-1"))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", response.Code)
	}
}

type httpWebhookAuditReader struct {
	attempts []webhookapplication.DeliveryAttempt
}

func (r *httpWebhookAuditReader) ListByJob(context.Context, string) ([]webhookapplication.DeliveryAttempt, error) {
	return r.attempts, nil
}

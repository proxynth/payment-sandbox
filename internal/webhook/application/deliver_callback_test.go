package application

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	schedulerdomain "proxynth/payment-sandbox/internal/scheduler/domain"
	webhookdomain "proxynth/payment-sandbox/internal/webhook/domain"
)

func TestOutboundCallbackExecutesJSONPost(t *testing.T) {
	endpoint, err := webhookdomain.NewEndpoint("endpoint-1", "https://example.test/hooks")
	if err != nil {
		t.Fatalf("NewEndpoint() error = %v", err)
	}
	client := &callbackClientFake{response: &http.Response{StatusCode: http.StatusAccepted, Body: io.NopCloser(bytes.NewBufferString("accepted"))}}
	delivery, err := NewOutboundCallback(&callbackRepositoryFake{endpoint: endpoint}, client)
	if err != nil {
		t.Fatalf("NewOutboundCallback() error = %v", err)
	}
	payload, err := NewDeliveryPayload(endpoint.ID(), []byte(`{"event":"payment.authorized"}`))
	if err != nil {
		t.Fatalf("NewDeliveryPayload() error = %v", err)
	}

	if err := delivery.Execute(context.Background(), payload); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if client.request.Method != http.MethodPost || client.request.URL.String() != endpoint.URL() {
		t.Fatalf("request = %s %s, want POST %s", client.request.Method, client.request.URL, endpoint.URL())
	}
	if got := client.request.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got, _ := io.ReadAll(client.request.Body); string(got) != `{"event":"payment.authorized"}` {
		t.Fatalf("request body = %q", got)
	}
	if !client.bodyClosed {
		t.Fatal("response body was not closed")
	}
}

func TestOutboundCallbackPropagatesContext(t *testing.T) {
	endpoint, _ := webhookdomain.NewEndpoint("endpoint-1", "https://example.test/hooks")
	client := &callbackClientFake{response: &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(nil))}}
	delivery, err := NewOutboundCallback(&callbackRepositoryFake{endpoint: endpoint}, client)
	if err != nil {
		t.Fatalf("NewOutboundCallback() error = %v", err)
	}
	payload, _ := NewDeliveryPayload(endpoint.ID(), []byte(`{}`))
	ctx := context.WithValue(context.Background(), callbackContextKey{}, "value")

	if err := delivery.Execute(ctx, payload); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := client.request.Context().Value(callbackContextKey{}); got != "value" {
		t.Fatalf("request context value = %v, want value", got)
	}
}

func TestOutboundCallbackPropagatesEventMetadata(t *testing.T) {
	endpoint, _ := webhookdomain.NewEndpoint("endpoint-1", "https://example.test/hooks")
	client := &callbackClientFake{response: &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(nil))}}
	delivery, err := NewOutboundCallback(&callbackRepositoryFake{endpoint: endpoint}, client)
	if err != nil {
		t.Fatalf("NewOutboundCallback() error = %v", err)
	}
	payload, err := NewDeliveryPayload(endpoint.ID(), []byte(`{}`), "request-1", "event-0")
	if err != nil {
		t.Fatalf("NewDeliveryPayload() error = %v", err)
	}
	if err := delivery.Execute(context.Background(), payload); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := client.request.Header.Get("X-Correlation-ID"); got != "request-1" {
		t.Fatalf("correlation header = %q", got)
	}
	if got := client.request.Header.Get("X-Causation-ID"); got != "event-0" {
		t.Fatalf("causation header = %q", got)
	}
}

func TestOutboundCallbackRejectsInvalidPayloadBeforeHTTP(t *testing.T) {
	client := &callbackClientFake{response: &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(nil))}}
	delivery, err := NewOutboundCallback(&callbackRepositoryFake{}, client)
	if err != nil {
		t.Fatalf("NewOutboundCallback() error = %v", err)
	}

	if err := delivery.Execute(context.Background(), []byte(`{"endpoint_id":"endpoint-1"}`)); !errors.Is(err, ErrInvalidDeliveryPayload) {
		t.Fatalf("Execute() error = %v, want %v", err, ErrInvalidDeliveryPayload)
	}
	if client.request != nil {
		t.Fatal("HTTP client was called for invalid payload")
	}
}

func TestOutboundCallbackMapsTransportAndHTTPFailures(t *testing.T) {
	endpoint, _ := webhookdomain.NewEndpoint("endpoint-1", "https://example.test/hooks")
	payload, _ := NewDeliveryPayload(endpoint.ID(), []byte(`{}`))

	transportErr := errors.New("connection refused")
	transportDelivery, err := NewOutboundCallback(&callbackRepositoryFake{endpoint: endpoint}, &callbackClientFake{err: transportErr})
	if err != nil {
		t.Fatalf("NewOutboundCallback() error = %v", err)
	}
	if err := transportDelivery.Execute(context.Background(), payload); !errors.Is(err, ErrCallbackDeliveryFailed) || !errors.Is(err, transportErr) {
		t.Fatalf("transport error = %v", err)
	}

	statusDelivery, err := NewOutboundCallback(&callbackRepositoryFake{endpoint: endpoint}, &callbackClientFake{
		response: &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(bytes.NewReader(nil))},
	})
	if err != nil {
		t.Fatalf("NewOutboundCallback() error = %v", err)
	}
	if err := statusDelivery.Execute(context.Background(), payload); !errors.Is(err, ErrCallbackDeliveryFailed) {
		t.Fatalf("status error = %v, want %v", err, ErrCallbackDeliveryFailed)
	}
}

// Invariant: each scheduler-owned delivery attempt records its observable
// transport outcome without retaining callback bodies.
func TestOutboundCallbackRecordsSchedulerAttemptOutcome(t *testing.T) {
	endpoint, _ := webhookdomain.NewEndpoint("endpoint-1", "https://example.test/hooks")
	audit := &callbackAuditFake{}
	delivery, err := NewOutboundCallbackWithAudit(
		&callbackRepositoryFake{endpoint: endpoint},
		&callbackClientFake{response: &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(bytes.NewReader(nil))}},
		audit,
	)
	if err != nil {
		t.Fatalf("NewOutboundCallbackWithAudit() error = %v", err)
	}
	payload, _ := NewDeliveryPayload(endpoint.ID(), []byte(`{}`), "request-1", "event-3")
	ctx := schedulerdomain.WithExecutionMetadata(context.Background(), schedulerdomain.ExecutionMetadata{JobID: "job-9", Attempt: 2})

	if err := delivery.Execute(ctx, payload); !errors.Is(err, ErrCallbackDeliveryFailed) {
		t.Fatalf("Execute() error = %v, want callback failure", err)
	}
	if len(audit.attempts) != 1 {
		t.Fatalf("audit records = %d, want 1", len(audit.attempts))
	}
	got := audit.attempts[0]
	if got.JobID != "job-9" || got.Attempt != 2 || got.EndpointID != endpoint.ID() || got.CorrelationID != "request-1" || got.CausationID != "event-3" {
		t.Fatalf("audit identity = %#v", got)
	}
	if got.Outcome != DeliveryFailed || got.HTTPStatus != http.StatusBadGateway || got.Error == "" {
		t.Fatalf("audit result = %#v", got)
	}
}

func TestOutboundCallbackRecordsSuccessfulOutcome(t *testing.T) {
	endpoint, _ := webhookdomain.NewEndpoint("endpoint-1", "https://example.test/hooks")
	audit := &callbackAuditFake{}
	delivery, err := NewOutboundCallbackWithAudit(
		&callbackRepositoryFake{endpoint: endpoint},
		&callbackClientFake{response: &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(bytes.NewReader(nil))}},
		audit,
	)
	if err != nil {
		t.Fatalf("NewOutboundCallbackWithAudit() error = %v", err)
	}
	payload, _ := NewDeliveryPayload(endpoint.ID(), []byte(`{}`))

	if err := delivery.Execute(context.Background(), payload); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(audit.attempts) != 1 {
		t.Fatalf("audit records = %d, want 1", len(audit.attempts))
	}
	got := audit.attempts[0]
	if got.Outcome != DeliverySucceeded || got.HTTPStatus != http.StatusNoContent || got.Error != "" || got.JobID != "direct:" {
		t.Fatalf("audit result = %#v", got)
	}
}

func TestNewDeliveryPayloadValidatesBody(t *testing.T) {
	if _, err := NewDeliveryPayload("endpoint-1", []byte(`not-json`)); !errors.Is(err, ErrInvalidDeliveryPayload) {
		t.Fatalf("NewDeliveryPayload() error = %v, want %v", err, ErrInvalidDeliveryPayload)
	}
}

type callbackContextKey struct{}

type callbackRepositoryFake struct {
	endpoint *webhookdomain.Endpoint
}

func (r *callbackRepositoryFake) Save(context.Context, *webhookdomain.Endpoint) error { return nil }

func (r *callbackRepositoryFake) FindByID(_ context.Context, _ webhookdomain.EndpointID) (*webhookdomain.Endpoint, error) {
	if r.endpoint == nil {
		return nil, ErrEndpointNotFound
	}
	return r.endpoint, nil
}

func (r *callbackRepositoryFake) List(context.Context) ([]*webhookdomain.Endpoint, error) {
	return nil, nil
}

type callbackClientFake struct {
	err        error
	response   *http.Response
	request    *http.Request
	bodyClosed bool
}

type callbackAuditFake struct {
	attempts []DeliveryAttempt
	err      error
}

func (a *callbackAuditFake) Record(_ context.Context, attempt DeliveryAttempt) error {
	a.attempts = append(a.attempts, attempt)
	return a.err
}

func (c *callbackClientFake) Do(request *http.Request) (*http.Response, error) {
	c.request = request
	if c.err != nil {
		return nil, c.err
	}
	c.response.Body = &closeTrackingBody{ReadCloser: c.response.Body, closed: &c.bodyClosed}
	return c.response, nil
}

type closeTrackingBody struct {
	io.ReadCloser
	closed *bool
}

func (b *closeTrackingBody) Close() error {
	*b.closed = true
	return b.ReadCloser.Close()
}

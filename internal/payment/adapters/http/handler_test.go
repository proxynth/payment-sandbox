package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"proxynth/payment-sandbox/internal/api"
	paymentapplication "proxynth/payment-sandbox/internal/payment/application"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
)

type contextKey struct{}

func TestHandler_ExecutesPaymentLifecycle(t *testing.T) {
	repository := newTestRepository()
	server := newTestServer(t, repository)

	create := doJSON(t, server, http.MethodPost, "/payments", `{"id":"payment-1","amount":10000,"currency":"EUR"}`)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", create.Code, http.StatusCreated, create.Body.String())
	}
	if location := create.Header().Get("Location"); location != "/payments/payment-1" {
		t.Fatalf("Location = %q, want %q", location, "/payments/payment-1")
	}

	get := doRequest(t, server, http.MethodGet, "/payments/payment-1", "", "")
	assertStatus(t, get, http.StatusOK)

	authorize := doRequest(t, server, http.MethodPost, "/payments/payment-1/authorize", "", "")
	assertStatus(t, authorize, http.StatusOK)

	capture := doJSON(t, server, http.MethodPost, "/payments/payment-1/capture", `{"amount":10000,"currency":"EUR"}`)
	assertStatus(t, capture, http.StatusOK)

	refund := doJSON(t, server, http.MethodPost, "/payments/payment-1/refund", `{"amount":10000,"currency":"EUR"}`)
	assertStatus(t, refund, http.StatusOK)

	var response struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(refund.Body).Decode(&response); err != nil {
		t.Fatalf("decode refund response: %v", err)
	}
	if response.Status != string(paymentdomain.StatusRefunded) {
		t.Fatalf("status = %q, want %q", response.Status, paymentdomain.StatusRefunded)
	}
}

func TestHandler_MapsErrorsToHTTPResponses(t *testing.T) {
	server := newTestServer(t, newTestRepository())

	missing := doRequest(t, server, http.MethodGet, "/payments/missing", "", "")
	assertStatus(t, missing, http.StatusNotFound)

	malformed := doJSON(t, server, http.MethodPost, "/payments", `{"id":`)
	assertStatus(t, malformed, http.StatusBadRequest)

	unsupportedMediaType := doRequest(t, server, http.MethodPost, "/payments", `{"id":"payment-1"}`, "text/plain")
	assertStatus(t, unsupportedMediaType, http.StatusUnsupportedMediaType)

	unknownField := doJSON(t, server, http.MethodPost, "/payments", `{"id":"payment-1","unknown":true}`)
	assertStatus(t, unknownField, http.StatusBadRequest)
}

func TestHandler_PropagatesRequestContext(t *testing.T) {
	repository := newTestRepository()
	server := newTestServer(t, repository)
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/payments", strings.NewReader(`{"id":"payment-1","amount":100,"currency":"EUR"}`))
	request = request.WithContext(context.WithValue(request.Context(), contextKey{}, "request-1"))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	assertStatus(t, recorder, http.StatusCreated)
	if repository.contextValue != "request-1" {
		t.Fatalf("repository context value = %q, want request-1", repository.contextValue)
	}
}

func TestHandler_GeneratesAndReturnsCorrelationID(t *testing.T) {
	server := newTestServer(t, newTestRepository())
	recorder := doJSON(t, server, http.MethodPost, "/payments", `{"id":"payment-1","amount":100,"currency":"EUR"}`)
	if recorder.Header().Get("X-Correlation-ID") == "" {
		t.Fatal("X-Correlation-ID response header is empty")
	}
}

func TestHandler_PreservesCorrelationID(t *testing.T) {
	server := newTestServer(t, newTestRepository())
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/payments", strings.NewReader(`{"id":"payment-1","amount":100,"currency":"EUR"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Correlation-ID", "request-1")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	assertStatus(t, recorder, http.StatusCreated)
	if got := recorder.Header().Get("X-Correlation-ID"); got != "request-1" {
		t.Fatalf("X-Correlation-ID = %q, want request-1", got)
	}
}

func TestHandler_RejectsInvalidCorrelationID(t *testing.T) {
	server := newTestServer(t, newTestRepository())
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/payments", strings.NewReader(`{"id":"payment-1","amount":100,"currency":"EUR"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Correlation-ID", strings.Repeat("x", 129))
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	assertStatus(t, recorder, http.StatusBadRequest)
}

func TestNewHandler_RejectsNilRepository(t *testing.T) {
	if _, err := NewHandler(nil); !errors.Is(err, ErrNilRepository) {
		t.Fatalf("NewHandler() error = %v, want %v", err, ErrNilRepository)
	}
}

type testRepository struct {
	payments     map[paymentdomain.ID]*paymentdomain.Payment
	contextValue string
}

func newTestRepository() *testRepository {
	return &testRepository{payments: make(map[paymentdomain.ID]*paymentdomain.Payment)}
}

func (r *testRepository) Save(ctx context.Context, payment *paymentdomain.Payment) error {
	if value, ok := ctx.Value(contextKey{}).(string); ok {
		r.contextValue = value
	}
	r.payments[payment.ID()] = payment
	return nil
}

func (r *testRepository) FindByID(ctx context.Context, id paymentdomain.ID) (*paymentdomain.Payment, error) {
	if value, ok := ctx.Value(contextKey{}).(string); ok {
		r.contextValue = value
	}
	payment, exists := r.payments[id]
	if !exists {
		return nil, paymentapplication.ErrPaymentNotFound
	}
	return payment, nil
}

func newTestServer(t *testing.T, repository paymentapplication.Repository) *api.Server {
	t.Helper()

	server, err := api.NewServer(":8080")
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	handler, err := NewHandler(repository)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	if err := handler.Register(server); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	return server
}

func doJSON(t *testing.T, server http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return doRequest(t, server, method, path, body, "application/json")
}

func doRequest(t *testing.T, server http.Handler, method, path, body, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

func assertStatus(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	if recorder.Code != want {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, want, recorder.Body.String())
	}
}

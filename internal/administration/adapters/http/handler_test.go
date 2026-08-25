package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/api"
	"proxynth/payment-sandbox/internal/platform/clock"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
)

func TestHandler_ExposesTimeAndProviders(t *testing.T) {
	initial := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	virtualClock, err := clock.NewVirtualClock(initial)
	if err != nil {
		t.Fatalf("NewVirtualClock() error = %v", err)
	}
	handler, err := NewHandler(virtualClock, providerRegistryFake{ids: []providerdomain.ProviderID{"adyen", "stripe"}})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	server := newServer(t, handler)

	current := serve(server, newRequest(t, http.MethodGet, "/admin/time", ""))
	if current.Code != http.StatusOK || !strings.Contains(current.Body.String(), "2026-01-01T12:00:00Z") {
		t.Fatalf("current response = %d %s", current.Code, current.Body.String())
	}

	advance := newRequest(t, http.MethodPost, "/admin/time/advance", `{"by":"5s"}`)
	advance.Header.Set("Content-Type", "application/json")
	advanced := serve(server, advance)
	if advanced.Code != http.StatusOK || !strings.Contains(advanced.Body.String(), "2026-01-01T12:00:05Z") {
		t.Fatalf("advanced response = %d %s", advanced.Code, advanced.Body.String())
	}

	providers := serve(server, newRequest(t, http.MethodGet, "/admin/providers", ""))
	if providers.Code != http.StatusOK || !strings.Contains(providers.Body.String(), `"providers":["adyen","stripe"]`) {
		t.Fatalf("providers response = %d %q", providers.Code, providers.Body.String())
	}
}

func TestHandler_RejectsInvalidAdvanceRequest(t *testing.T) {
	virtualClock, err := clock.NewVirtualClock(time.Now())
	if err != nil {
		t.Fatalf("NewVirtualClock() error = %v", err)
	}
	handler, err := NewHandler(virtualClock, providerRegistryFake{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	server := newServer(t, handler)

	request := newRequest(t, http.MethodPost, "/admin/time/advance", `{"by":"-1s"}`)
	request.Header.Set("Content-Type", "application/json")
	response := serve(server, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}

func TestHandler_MapsClockErrors(t *testing.T) {
	handler, err := NewHandler(failingClock{}, providerRegistryFake{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	server := newServer(t, handler)

	request := newRequest(t, http.MethodPost, "/admin/time/advance", `{"by":"1s"}`)
	request.Header.Set("Content-Type", "application/json")
	response := serve(server, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusConflict, response.Body.String())
	}
}

func TestNewHandler_RejectsNilDependencies(t *testing.T) {
	if _, err := NewHandler(nil, providerRegistryFake{}); !errors.Is(err, ErrNilClock) {
		t.Fatalf("nil clock error = %v, want %v", err, ErrNilClock)
	}
	virtualClock, err := clock.NewVirtualClock(time.Now())
	if err != nil {
		t.Fatalf("NewVirtualClock() error = %v", err)
	}
	if _, err := NewHandler(virtualClock, nil); !errors.Is(err, ErrNilRegistry) {
		t.Fatalf("nil registry error = %v, want %v", err, ErrNilRegistry)
	}
}

func newServer(t *testing.T, handler *Handler) *api.Server {
	t.Helper()
	server, err := api.NewServer(":8080")
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if err := handler.Register(server, "test-admin-token"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	return server
}

type providerRegistryFake struct {
	ids []providerdomain.ProviderID
}

func (f providerRegistryFake) IDs() []providerdomain.ProviderID { return f.ids }

type failingClock struct{}

func (failingClock) Now() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

func (failingClock) Advance(time.Duration) error { return clock.ErrBackwardAdvance }

func newRequest(t *testing.T, method, path, body string) *http.Request {
	t.Helper()
	request := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer test-admin-token")
	return request
}

func adminRequest(method, path string) *http.Request {
	request := httptest.NewRequestWithContext(context.Background(), method, path, nil)
	request.Header.Set("Authorization", "Bearer test-admin-token")
	return request
}

func serve(server *api.Server, request *http.Request) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	return response
}

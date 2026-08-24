package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"proxynth/payment-sandbox/internal/api"
	webhookapplication "proxynth/payment-sandbox/internal/webhook/application"
	webhookdomain "proxynth/payment-sandbox/internal/webhook/domain"
)

func TestHandler_RegistersGetsAndListsEndpoints(t *testing.T) {
	repository := newRepository()
	handler, err := NewHandler(repository)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	server := newServer(t, handler)

	create := newRequest(t, http.MethodPost, "/webhook-endpoints", `{"id":"endpoint-1","url":"https://example.test/hooks"}`)
	create.Header.Set("Content-Type", "application/json")
	created := serve(server, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body = %s", created.Code, http.StatusCreated, created.Body.String())
	}
	if got := created.Header().Get("Location"); got != "/webhook-endpoints/endpoint-1" {
		t.Errorf("Location = %q", got)
	}

	got := serve(server, newRequest(t, http.MethodGet, "/webhook-endpoints/endpoint-1", ""))
	if got.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d; body = %s", got.Code, http.StatusOK, got.Body.String())
	}
	for _, expected := range []string{`"id":"endpoint-1"`, `"url":"https://example.test/hooks"`, `"enabled":true`} {
		if !strings.Contains(got.Body.String(), expected) {
			t.Errorf("get body = %q, missing %q", got.Body.String(), expected)
		}
	}

	list := serve(server, newRequest(t, http.MethodGet, "/webhook-endpoints", ""))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "endpoint-1") {
		t.Fatalf("list response = %d %s", list.Code, list.Body.String())
	}
}

func TestHandler_RejectsInvalidEndpointRequest(t *testing.T) {
	handler, err := NewHandler(newRepository())
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	server := newServer(t, handler)
	request := newRequest(t, http.MethodPost, "/webhook-endpoints", `{"id":"endpoint-1","url":"not-a-url"}`)
	request.Header.Set("Content-Type", "application/json")
	response := serve(server, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}

func TestHandler_MapsRepositoryErrors(t *testing.T) {
	repository := newRepository()
	repository.findErr = webhookapplication.ErrEndpointNotFound
	handler, err := NewHandler(repository)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	server := newServer(t, handler)
	response := serve(server, newRequest(t, http.MethodGet, "/webhook-endpoints/endpoint-1", ""))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusNotFound, response.Body.String())
	}
}

func TestHandler_PropagatesRequestContext(t *testing.T) {
	repository := newRepository()
	handler, err := NewHandler(repository)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	server := newServer(t, handler)
	request := newRequest(t, http.MethodPost, "/webhook-endpoints", `{"id":"endpoint-1","url":"https://example.test/hooks"}`)
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(context.WithValue(request.Context(), contextKey("request"), "value"))
	response := serve(server, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if repository.contextValue != "value" {
		t.Errorf("contextValue = %q, want %q", repository.contextValue, "value")
	}
}

func TestNewHandler_RejectsNilRepository(t *testing.T) {
	if _, err := NewHandler(nil); !errors.Is(err, ErrNilRepository) {
		t.Fatalf("NewHandler() error = %v, want %v", err, ErrNilRepository)
	}
}

func newServer(t *testing.T, handler *Handler) *api.Server {
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

type contextKey string

type repository struct {
	items        map[webhookdomain.EndpointID]*webhookdomain.Endpoint
	findErr      error
	contextValue any
}

func newRepository() *repository {
	return &repository{items: make(map[webhookdomain.EndpointID]*webhookdomain.Endpoint)}
}

func (r *repository) Save(ctx context.Context, endpoint *webhookdomain.Endpoint) error {
	if value := ctx.Value(contextKey("request")); value != nil {
		r.contextValue = value
	}
	if _, exists := r.items[endpoint.ID()]; exists {
		return webhookapplication.ErrEndpointAlreadyExists
	}
	r.items[endpoint.ID()] = endpoint
	return nil
}

func (r *repository) FindByID(_ context.Context, id webhookdomain.EndpointID) (*webhookdomain.Endpoint, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	endpoint, exists := r.items[id]
	if !exists {
		return nil, webhookapplication.ErrEndpointNotFound
	}
	return endpoint, nil
}

func (r *repository) List(_ context.Context) ([]*webhookdomain.Endpoint, error) {
	result := make([]*webhookdomain.Endpoint, 0, len(r.items))
	for _, endpoint := range r.items {
		result = append(result, endpoint)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID() < result[j].ID() })
	return result, nil
}

func newRequest(t *testing.T, method, path, body string) *http.Request {
	t.Helper()
	return httptest.NewRequestWithContext(
		context.Background(),
		method,
		path,
		strings.NewReader(body),
	)
}

func serve(server *api.Server, request *http.Request) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	return response
}

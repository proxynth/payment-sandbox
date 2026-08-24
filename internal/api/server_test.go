package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type requestIDKey struct{}

func TestServer_HealthEndpoints(t *testing.T) {
	server, err := NewServer(":8080")
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   HealthResponse
	}{
		{name: "live", path: "/health/live", wantStatus: http.StatusOK, wantBody: HealthResponse{Status: "alive"}},
		{name: "ready", path: "/health/ready", wantStatus: http.StatusOK, wantBody: HealthResponse{Status: "ready"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, tt.path, nil)
			recorder := httptest.NewRecorder()

			server.Handler().ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", contentType)
			}

			var got HealthResponse
			if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got != tt.wantBody {
				t.Fatalf("body = %+v, want %+v", got, tt.wantBody)
			}
		})
	}
}

func TestServer_ReadinessCanBeChanged(t *testing.T) {
	server, err := NewServer(":8080")
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	server.SetReady(false)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health/ready", nil),
	)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}

	var got HealthResponse
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Status != "not_ready" {
		t.Fatalf("status body = %q, want not_ready", got.Status)
	}
}

func TestRouter_HandlesRoutesAndProtocolErrors(t *testing.T) {
	router := NewRouter()
	if err := router.Handle(http.MethodGet, "/resource", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Context().Value(requestIDKey{}) != "req-1" {
			t.Error("request context was not propagated")
		}
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
	})); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	successRecorder := httptest.NewRecorder()
	successRequest := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/resource", nil)
	successRequest = successRequest.WithContext(context.WithValue(successRequest.Context(), requestIDKey{}, "req-1"))
	router.ServeHTTP(successRecorder, successRequest)
	if successRecorder.Code != http.StatusOK {
		t.Fatalf("successful route status = %d, want %d", successRecorder.Code, http.StatusOK)
	}

	tests := []struct {
		name          string
		method        string
		path          string
		wantStatus    int
		wantAllow     string
		wantErrorCode string
	}{
		{name: "unknown path", method: http.MethodGet, path: "/unknown", wantStatus: http.StatusNotFound, wantErrorCode: "not_found"},
		{name: "unsupported method", method: http.MethodPost, path: "/resource", wantStatus: http.StatusMethodNotAllowed, wantAllow: http.MethodGet, wantErrorCode: "method_not_allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(context.Background(), tt.method, tt.path, nil)
			request = request.WithContext(context.WithValue(request.Context(), requestIDKey{}, "req-1"))

			router.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if got := recorder.Header().Get("Allow"); got != tt.wantAllow {
				t.Fatalf("Allow = %q, want %q", got, tt.wantAllow)
			}

			var response ErrorResponse
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if response.Error.Code != tt.wantErrorCode {
				t.Fatalf("error code = %q, want %q", response.Error.Code, tt.wantErrorCode)
			}
			if !strings.Contains(response.Error.Message, "route") && tt.name == "unknown path" {
				t.Fatalf("error message = %q, want route context", response.Error.Message)
			}
		})
	}
}

func TestNewServer_RejectsEmptyAddress(t *testing.T) {
	if _, err := NewServer(""); !errors.Is(err, ErrInvalidAddress) {
		t.Fatalf("NewServer() error = %v, want %v", err, ErrInvalidAddress)
	}
}

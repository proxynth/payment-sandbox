package domain

import (
	"errors"
	"testing"
)

func TestNewEndpoint_ValidatesIdentityAndURL(t *testing.T) {
	tests := []struct {
		name string
		id   EndpointID
		url  string
		err  error
	}{
		{name: "valid https", id: "endpoint-1", url: "https://example.test/hooks"},
		{name: "valid http", id: "endpoint-1", url: "http://localhost:8080/hooks"},
		{name: "missing id", url: "https://example.test/hooks", err: ErrInvalidEndpointID},
		{name: "missing scheme", id: "endpoint-1", url: "example.test/hooks", err: ErrInvalidEndpointURL},
		{name: "unsupported scheme", id: "endpoint-1", url: "ftp://example.test/hooks", err: ErrInvalidEndpointURL},
		{name: "missing host", id: "endpoint-1", url: "https:///hooks", err: ErrInvalidEndpointURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint, err := NewEndpoint(tt.id, tt.url)
			if !errors.Is(err, tt.err) {
				t.Fatalf("NewEndpoint() error = %v, want %v", err, tt.err)
			}
			if tt.err == nil && (!endpoint.Enabled() || endpoint.ID() != tt.id || endpoint.URL() != tt.url) {
				t.Fatalf("endpoint = %#v", endpoint)
			}
		})
	}
}

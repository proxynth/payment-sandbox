package client

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"testing"
)

func TestNewDisablesProxyAndRedirects(t *testing.T) {
	client := New()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("webhook client must not use an HTTP proxy")
	}
	if err := client.CheckRedirect(nil, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("CheckRedirect() error = %v, want %v", err, http.ErrUseLastResponse)
	}
}

func TestSafeDialContextRejectsPrivateDestinations(t *testing.T) {
	for _, address := range []string{"localhost:8080", "127.0.0.1:8080", "10.0.0.1:8080", "[::1]:8080"} {
		t.Run(address, func(t *testing.T) {
			_, err := safeDialContext(context.Background(), "tcp", address)
			if err == nil || !strings.Contains(err.Error(), "public") {
				t.Fatalf("safeDialContext() error = %v, want non-public destination error", err)
			}
		})
	}
}

func TestIsPublicAddress(t *testing.T) {
	for _, test := range []struct {
		name   string
		input  string
		public bool
	}{
		{name: "public", input: "8.8.8.8", public: true},
		{name: "private v4", input: "192.168.1.10"},
		{name: "private v6", input: "fd00::1"},
		{name: "loopback", input: "127.0.0.1"},
		{name: "link local", input: "169.254.1.1"},
		{name: "unspecified", input: "::"},
	} {
		t.Run(test.name, func(t *testing.T) {
			address := net.ParseIP(test.input)
			parsed, ok := netip.AddrFromSlice(address)
			if !ok || isPublicAddress(parsed) != test.public {
				t.Fatalf("isPublicAddress(%s) = %v, want %v", test.input, ok && isPublicAddress(parsed), test.public)
			}
		})
	}
}

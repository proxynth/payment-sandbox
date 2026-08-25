package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"
)

// New returns the runtime HTTP client used for outbound webhook delivery.
// It validates every DNS resolution at connection time and never follows
// redirects, preventing a public endpoint from turning into an SSRF proxy.
func New() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy:                 nil,
			DialContext:           safeDialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			IdleConnTimeout:       30 * time.Second,
		},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func safeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("parse webhook destination %q: %w", address, err)
	}
	if strings.EqualFold(strings.TrimSuffix(host, "."), "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return nil, fmt.Errorf("webhook destination %q is not public", host)
	}

	resolved, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve webhook destination %q: %w", host, err)
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf("resolve webhook destination %q: no addresses", host)
	}

	addresses := make([]netip.Addr, 0, len(resolved))
	for _, resolvedIP := range resolved {
		parsed, ok := netip.AddrFromSlice(resolvedIP.IP)
		if !ok || !isPublicAddress(parsed) {
			return nil, fmt.Errorf("webhook destination %q resolves to a non-public address", host)
		}
		addresses = append(addresses, parsed)
	}

	dialer := net.Dialer{}
	var lastErr error
	for _, resolvedAddress := range addresses {
		connection, err := dialer.DialContext(ctx, network, net.JoinHostPort(resolvedAddress.String(), port))
		if err == nil {
			return connection, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func isPublicAddress(address netip.Addr) bool {
	address = address.Unmap()
	return address.IsValid() && !address.IsPrivate() && !address.IsLoopback() &&
		!address.IsLinkLocalUnicast() && !address.IsLinkLocalMulticast() &&
		!address.IsMulticast() && !address.IsUnspecified()
}

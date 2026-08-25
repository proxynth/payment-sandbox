package domain

import (
	"net/netip"
	"net/url"
	"strings"
)

type EndpointID string

type Endpoint struct {
	id      EndpointID
	url     string
	enabled bool
}

func NewEndpoint(id EndpointID, endpointURL string) (*Endpoint, error) {
	if id == "" {
		return nil, ErrInvalidEndpointID
	}

	if !validURL(endpointURL) {
		return nil, ErrInvalidEndpointURL
	}

	return &Endpoint{id: id, url: endpointURL, enabled: true}, nil
}

func (e *Endpoint) ID() EndpointID { return e.id }

func (e *Endpoint) URL() string { return e.url }

func (e *Endpoint) Enabled() bool { return e.enabled }

func validURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return false
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}

	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return false
	}

	address, err := netip.ParseAddr(host)
	if err != nil {
		return true
	}
	return isPublicAddress(address)
}

func isPublicAddress(address netip.Addr) bool {
	address = address.Unmap()
	return address.IsValid() && !address.IsPrivate() && !address.IsLoopback() &&
		!address.IsLinkLocalUnicast() && !address.IsLinkLocalMulticast() &&
		!address.IsMulticast() && !address.IsUnspecified()
}

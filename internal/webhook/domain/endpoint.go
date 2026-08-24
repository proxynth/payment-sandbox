package domain

import (
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
	if err != nil || parsed.Host == "" {
		return false
	}

	scheme := strings.ToLower(parsed.Scheme)
	return scheme == "http" || scheme == "https"
}

package application

import "errors"

var (
	ErrInvalidRepository      = errors.New("invalid webhook endpoint repository")
	ErrEndpointNotFound       = errors.New("webhook endpoint not found")
	ErrEndpointAlreadyExists  = errors.New("webhook endpoint already exists")
	ErrInvalidDeliveryPayload = errors.New("invalid webhook delivery payload")
	ErrCallbackDeliveryFailed = errors.New("webhook callback delivery failed")
	ErrInvalidHTTPClient      = errors.New("invalid webhook HTTP client")
)

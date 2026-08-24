package application

import "errors"

var (
	ErrEndpointNotFound      = errors.New("webhook endpoint not found")
	ErrEndpointAlreadyExists = errors.New("webhook endpoint already exists")
)

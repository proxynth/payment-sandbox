package domain

import "errors"

var (
	ErrInvalidEndpointID  = errors.New("invalid webhook endpoint id")
	ErrInvalidEndpointURL = errors.New("invalid webhook endpoint URL")
)

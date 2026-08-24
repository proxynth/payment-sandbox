package api

import "errors"

var (
	ErrInvalidAddress = errors.New("invalid HTTP server address")
	ErrInvalidMethod  = errors.New("invalid HTTP method")
	ErrInvalidPath    = errors.New("invalid HTTP path")
	ErrNilHandler     = errors.New("nil HTTP handler")
	ErrDuplicateRoute = errors.New("duplicate HTTP route")
)

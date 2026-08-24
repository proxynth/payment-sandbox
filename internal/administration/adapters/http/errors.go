package http

import "errors"

var (
	ErrNilClock    = errors.New("administration clock is nil")
	ErrNilRegistry = errors.New("administration provider registry is nil")
)

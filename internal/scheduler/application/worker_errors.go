package application

import "errors"

var (
	ErrInvalidWorkerRepository = errors.New("invalid worker repository")
	ErrNilJob                  = errors.New("worker received a nil job")
	ErrUnknownJobType          = errors.New("unknown job type")
	ErrInvalidHandler          = errors.New("invalid job handler")
)

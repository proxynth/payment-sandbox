package application

import "errors"

var (
	ErrInvalidRepository    = errors.New("invalid scheduler repository")
	ErrInvalidDispatcher    = errors.New("invalid scheduler dispatcher")
	ErrInvalidClock         = errors.New("invalid scheduler clock")
	ErrInvalidOwner         = errors.New("invalid scheduler owner")
	ErrInvalidBatchSize     = errors.New("invalid scheduler batch size")
	ErrInvalidLeaseDuration = errors.New("invalid scheduler lease duration")
	ErrNilAcquiredJob       = errors.New("scheduler acquired a nil job")
)

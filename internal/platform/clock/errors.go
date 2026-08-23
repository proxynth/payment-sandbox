package clock

import "errors"

var (
	ErrInvalidClock    = errors.New("invalid clock")
	ErrInvalidTime     = errors.New("invalid clock time")
	ErrInvalidAdvance  = errors.New("invalid clock advance")
	ErrBackwardAdvance = errors.New("clock cannot move backwards")
)

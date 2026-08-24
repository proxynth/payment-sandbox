package deterministic

import "errors"

var (
	ErrInvalidRequest         = errors.New("invalid provider request")
	ErrInvalidOperationAmount = errors.New("invalid provider operation amount")
)

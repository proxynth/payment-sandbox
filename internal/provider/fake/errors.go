package fake

import "errors"

var (
	ErrInvalidRequest         = errors.New("invalid fake provider request")
	ErrInvalidOperationAmount = errors.New("invalid fake provider operation amount")
)

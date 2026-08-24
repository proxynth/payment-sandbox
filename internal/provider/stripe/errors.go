package stripe

import "errors"

var (
	ErrInvalidRequest         = errors.New("invalid stripe provider request")
	ErrInvalidOperationAmount = errors.New("invalid stripe provider operation amount")
)

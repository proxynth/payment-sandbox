package stripe

import "proxynth/payment-sandbox/internal/provider/deterministic"

var (
	ErrInvalidRequest         = deterministic.ErrInvalidRequest
	ErrInvalidOperationAmount = deterministic.ErrInvalidOperationAmount
)

package domain

import "errors"

var (
	ErrInvalidProviderID      = errors.New("invalid provider id")
	ErrInvalidPaymentSnapshot = errors.New("invalid payment snapshot")
	ErrInvalidOperationResult = errors.New("invalid operation result")
)

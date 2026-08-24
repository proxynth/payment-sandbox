package domain

import "errors"

var (
	ErrInvalidProviderID      = errors.New("invalid provider id")
	ErrInvalidPaymentSnapshot = errors.New("invalid payment snapshot")
	ErrInvalidOperationResult = errors.New("invalid operation result")
	ErrInvalidAsyncOperation  = errors.New("invalid asynchronous operation")
	ErrNilProvider            = errors.New("nil provider")
	ErrProviderAlreadyExists  = errors.New("provider already registered")
	ErrProviderNotFound       = errors.New("provider not found")
)

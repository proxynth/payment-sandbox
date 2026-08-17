package domain

import "errors"

var (
	ErrInvalidPaymentID      = errors.New("invalid payment id")
	ErrInvalidPaymentStatus  = errors.New("invalid payment status")
	ErrInvalidPaymentVersion = errors.New("invalid payment version")
)

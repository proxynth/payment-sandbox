package domain

import "errors"

var (
	ErrInvalidPaymentID        = errors.New("invalid payment id")
	ErrInvalidPaymentStatus    = errors.New("invalid payment status")
	ErrInvalidPaymentVersion   = errors.New("invalid payment version")
	ErrInvalidMoneyAmount      = errors.New("invalid money amount")
	ErrInvalidCurrency         = errors.New("invalid currency")
	ErrInvalidTransition       = errors.New("invalid payment transition")
	ErrInvalidAuthorizedAmount = errors.New("invalid authorized amount")
	ErrInvalidCapturedAmount   = errors.New("invalid captured amount")
	ErrInvalidRefundedAmount   = errors.New("invalid refunded amount")
	ErrInvalidPaymentState     = errors.New("invalid payment state")
	ErrInvalidEventID          = errors.New("invalid event id")
	ErrInvalidAggregateID      = errors.New("invalid aggregate id")
	ErrInvalidEventType        = errors.New("invalid event type")
	ErrInvalidEventTimestamp   = errors.New("invalid event timestamp")
	ErrInvalidAggregateVersion = errors.New("invalid aggregate version")
)

package domain

import "errors"

var (
	ErrInvalidJobID           = errors.New("invalid job id")
	ErrInvalidJobType         = errors.New("invalid job type")
	ErrInvalidScheduledAt     = errors.New("invalid scheduled time")
	ErrInvalidLeaseOwner      = errors.New("invalid lease owner")
	ErrInvalidLeaseExpiry     = errors.New("invalid lease expiry")
	ErrInvalidRetryTime       = errors.New("invalid retry time")
	ErrInvalidJobTransition   = errors.New("invalid job transition")
	ErrInvalidExecutionStatus = errors.New("invalid job execution status")
	ErrInvalidAttempt         = errors.New("invalid retry attempt")
	ErrInvalidMaxAttempts     = errors.New("invalid maximum attempts")
	ErrInvalidRetryDelay      = errors.New("invalid retry delay")
	ErrInvalidMaxDelay        = errors.New("invalid maximum retry delay")
)

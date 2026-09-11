package application

import "errors"

var (
	ErrNotFound            = errors.New("idempotency record not found")
	ErrFingerprintConflict = errors.New("idempotency fingerprint conflict")
)

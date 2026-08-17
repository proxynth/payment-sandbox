package application

import "errors"

var (
	ErrPaymentNotFound        = errors.New("payment not found")
	ErrPaymentVersionConflict = errors.New("payment version conflict")
)

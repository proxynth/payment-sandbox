package application

import "errors"

var (
	ErrPaymentAlreadyExists    = errors.New("payment already exists")
	ErrNilScenarioRunner       = errors.New("nil scenario runner")
	ErrNilProviderRegistry     = errors.New("nil provider registry")
	ErrProviderOperationFailed = errors.New("provider operation failed")
	ErrAsyncOperationNotFound  = errors.New("asynchronous operation not found")
	ErrAsyncOperationNotDue    = errors.New("asynchronous operation is not due")
	ErrScenarioNotFound        = errors.New("scenario not found")
)

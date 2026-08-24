package application

import "errors"

var (
	ErrPaymentAlreadyExists = errors.New("payment already exists")
	ErrNilScenarioRunner    = errors.New("nil scenario runner")
	ErrNilProviderRegistry  = errors.New("nil provider registry")
)

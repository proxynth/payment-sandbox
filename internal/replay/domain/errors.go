package domain

import "errors"

var (
	ErrInvalidScenarioID            = errors.New("invalid scenario id")
	ErrInvalidScenarioTime          = errors.New("invalid scenario time")
	ErrInvalidProviderConfiguration = errors.New("invalid provider configuration")
	ErrDuplicateInitialPayment      = errors.New("duplicate initial payment")
	ErrInvalidCommand               = errors.New("invalid scenario command")
	ErrInvalidCommandType           = errors.New("invalid scenario command type")
	ErrInvalidCommandPaymentID      = errors.New("invalid scenario command payment id")
	ErrScenarioAlreadyExists        = errors.New("scenario already exists")
	ErrInvalidCommandAmount         = errors.New("invalid scenario command amount")
)

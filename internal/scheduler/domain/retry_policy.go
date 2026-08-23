package domain

import "time"

type RetryDecision struct {
	RetryAfter time.Duration
	Retry      bool
}

type RetryPolicy interface {
	Decide(attempt uint64) (RetryDecision, error)
}

type NoRetryPolicy struct{}

func NewNoRetryPolicy() NoRetryPolicy {
	return NoRetryPolicy{}
}

func (NoRetryPolicy) Decide(attempt uint64) (RetryDecision, error) {
	if attempt == 0 {
		return RetryDecision{}, ErrInvalidAttempt
	}

	return RetryDecision{}, nil
}

type FixedDelayPolicy struct {
	maxAttempts uint64
	delay       time.Duration
}

func NewFixedDelayPolicy(maxAttempts uint64, delay time.Duration) (FixedDelayPolicy, error) {
	if maxAttempts == 0 {
		return FixedDelayPolicy{}, ErrInvalidMaxAttempts
	}

	if delay <= 0 {
		return FixedDelayPolicy{}, ErrInvalidRetryDelay
	}

	return FixedDelayPolicy{
		maxAttempts: maxAttempts,
		delay:       delay,
	}, nil
}

func (p FixedDelayPolicy) Decide(attempt uint64) (RetryDecision, error) {
	if attempt == 0 {
		return RetryDecision{}, ErrInvalidAttempt
	}

	if attempt >= p.maxAttempts {
		return RetryDecision{}, nil
	}

	return RetryDecision{RetryAfter: p.delay, Retry: true}, nil
}

type ExponentialBackoffPolicy struct {
	maxAttempts uint64
	initial     time.Duration
	maxDelay    time.Duration
}

func NewExponentialBackoffPolicy(
	maxAttempts uint64,
	initial time.Duration,
	maxDelay time.Duration,
) (ExponentialBackoffPolicy, error) {
	if maxAttempts == 0 {
		return ExponentialBackoffPolicy{}, ErrInvalidMaxAttempts
	}

	if initial <= 0 {
		return ExponentialBackoffPolicy{}, ErrInvalidRetryDelay
	}

	if maxDelay < initial {
		return ExponentialBackoffPolicy{}, ErrInvalidMaxDelay
	}

	return ExponentialBackoffPolicy{
		maxAttempts: maxAttempts,
		initial:     initial,
		maxDelay:    maxDelay,
	}, nil
}

func (p ExponentialBackoffPolicy) Decide(attempt uint64) (RetryDecision, error) {
	if attempt == 0 {
		return RetryDecision{}, ErrInvalidAttempt
	}

	if attempt >= p.maxAttempts {
		return RetryDecision{}, nil
	}

	delay := p.initial
	for retryNumber := uint64(1); retryNumber < attempt; retryNumber++ {
		if delay > p.maxDelay/2 {
			return RetryDecision{RetryAfter: p.maxDelay, Retry: true}, nil
		}

		delay *= 2
	}

	return RetryDecision{RetryAfter: delay, Retry: true}, nil
}

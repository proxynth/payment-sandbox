package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNoRetryPolicy(t *testing.T) {
	policy := NewNoRetryPolicy()
	decision, err := policy.Decide(1)
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}

	if decision.Retry || decision.RetryAfter != 0 {
		t.Fatalf("decision = %+v, want terminal decision", decision)
	}
}

func TestFixedDelayPolicy(t *testing.T) {
	policy, err := NewFixedDelayPolicy(3, 30*time.Second)
	if err != nil {
		t.Fatalf("NewFixedDelayPolicy() error = %v", err)
	}

	for _, attempt := range []uint64{1, 2} {
		decision, err := policy.Decide(attempt)
		if err != nil {
			t.Fatalf("Decide(%d) error = %v", attempt, err)
		}
		if !decision.Retry || decision.RetryAfter != 30*time.Second {
			t.Fatalf("Decide(%d) = %+v, want retry after 30s", attempt, decision)
		}
	}

	decision, err := policy.Decide(3)
	if err != nil {
		t.Fatalf("Decide(3) error = %v", err)
	}
	if decision.Retry {
		t.Fatalf("Decide(3) = %+v, want terminal decision", decision)
	}
}

func TestExponentialBackoffPolicy(t *testing.T) {
	policy, err := NewExponentialBackoffPolicy(5, 5*time.Second, 30*time.Second)
	if err != nil {
		t.Fatalf("NewExponentialBackoffPolicy() error = %v", err)
	}

	wantDelays := map[uint64]time.Duration{
		1: 5 * time.Second,
		2: 10 * time.Second,
		3: 20 * time.Second,
		4: 30 * time.Second,
	}
	for attempt, wantDelay := range wantDelays {
		decision, err := policy.Decide(attempt)
		if err != nil {
			t.Fatalf("Decide(%d) error = %v", attempt, err)
		}
		if !decision.Retry || decision.RetryAfter != wantDelay {
			t.Fatalf("Decide(%d) = %+v, want retry after %v", attempt, decision, wantDelay)
		}
	}
}

func TestExponentialBackoffPolicy_CapsBeforeOverflow(t *testing.T) {
	policy, err := NewExponentialBackoffPolicy(64, time.Hour, 24*time.Hour)
	if err != nil {
		t.Fatalf("NewExponentialBackoffPolicy() error = %v", err)
	}

	decision, err := policy.Decide(63)
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if !decision.Retry || decision.RetryAfter != 24*time.Hour {
		t.Fatalf("Decide(63) = %+v, want capped retry", decision)
	}
}

func TestRetryPolicies_RejectInvalidConfigurationAndAttempts(t *testing.T) {
	if _, err := NewFixedDelayPolicy(0, time.Second); !errors.Is(err, ErrInvalidMaxAttempts) {
		t.Fatalf("NewFixedDelayPolicy() error = %v, want %v", err, ErrInvalidMaxAttempts)
	}
	if _, err := NewFixedDelayPolicy(1, 0); !errors.Is(err, ErrInvalidRetryDelay) {
		t.Fatalf("NewFixedDelayPolicy() error = %v, want %v", err, ErrInvalidRetryDelay)
	}
	if _, err := NewExponentialBackoffPolicy(0, time.Second, time.Second); !errors.Is(err, ErrInvalidMaxAttempts) {
		t.Fatalf("NewExponentialBackoffPolicy() error = %v, want %v", err, ErrInvalidMaxAttempts)
	}
	if _, err := NewExponentialBackoffPolicy(1, 0, time.Second); !errors.Is(err, ErrInvalidRetryDelay) {
		t.Fatalf("NewExponentialBackoffPolicy() error = %v, want %v", err, ErrInvalidRetryDelay)
	}
	if _, err := NewExponentialBackoffPolicy(1, 2*time.Second, time.Second); !errors.Is(err, ErrInvalidMaxDelay) {
		t.Fatalf("NewExponentialBackoffPolicy() error = %v, want %v", err, ErrInvalidMaxDelay)
	}

	policies := []RetryPolicy{
		NewNoRetryPolicy(),
		mustFixedPolicy(t, 2, time.Second),
		mustExponentialPolicy(t, 2, time.Second, 2*time.Second),
	}
	for _, policy := range policies {
		if _, err := policy.Decide(0); !errors.Is(err, ErrInvalidAttempt) {
			t.Fatalf("Decide(0) error = %v, want %v", err, ErrInvalidAttempt)
		}
	}
}

func mustFixedPolicy(t *testing.T, maxAttempts uint64, delay time.Duration) FixedDelayPolicy {
	t.Helper()

	policy, err := NewFixedDelayPolicy(maxAttempts, delay)
	if err != nil {
		t.Fatalf("NewFixedDelayPolicy() error = %v", err)
	}

	return policy
}

func mustExponentialPolicy(t *testing.T, maxAttempts uint64, initial, maxDelay time.Duration) ExponentialBackoffPolicy {
	t.Helper()

	policy, err := NewExponentialBackoffPolicy(maxAttempts, initial, maxDelay)
	if err != nil {
		t.Fatalf("NewExponentialBackoffPolicy() error = %v", err)
	}

	return policy
}

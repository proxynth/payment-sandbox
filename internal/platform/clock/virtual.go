package clock

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
}

// SystemClock exposes wall-clock time for operational concerns such as lease
// expiry. Business behaviour should use VirtualClock instead.
type SystemClock struct{}

func NewSystemClock() SystemClock { return SystemClock{} }

func (SystemClock) Now() time.Time { return time.Now().UTC() }

type VirtualClock struct {
	mu      sync.RWMutex
	current time.Time
}

func NewVirtualClock(initial time.Time) (*VirtualClock, error) {
	if initial.IsZero() {
		return nil, ErrInvalidTime
	}

	return &VirtualClock{current: initial.UTC()}, nil
}

func (c *VirtualClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.current
}

func (c *VirtualClock) Advance(by time.Duration) error {
	if by <= 0 {
		return ErrInvalidAdvance
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	next := c.current.Add(by)
	if !next.After(c.current) {
		return ErrBackwardAdvance
	}

	c.current = next
	return nil
}

func (c *VirtualClock) Restore(at time.Time) error {
	if at.IsZero() {
		return ErrInvalidTime
	}

	c.mu.Lock()
	c.current = at.UTC()
	c.mu.Unlock()

	return nil
}

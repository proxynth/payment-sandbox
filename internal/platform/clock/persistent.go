package clock

import (
	"context"
	"sync"
	"time"
)

type StateStore interface {
	Load(context.Context, string) (time.Time, bool, error)
	Save(context.Context, string, time.Time) error
}

type PersistentVirtualClock struct {
	mu      sync.RWMutex
	current time.Time
	store   StateStore
	key     string
}

func NewPersistentVirtualClock(initial time.Time, store StateStore, key string) (*PersistentVirtualClock, error) {
	if initial.IsZero() || store == nil || key == "" {
		return nil, ErrInvalidTime
	}
	at, exists, err := store.Load(context.Background(), key)
	if err != nil {
		return nil, err
	}
	if !exists {
		at = initial.UTC()
		if err := store.Save(context.Background(), key, at); err != nil {
			return nil, err
		}
	}
	return &PersistentVirtualClock{current: at.UTC(), store: store, key: key}, nil
}

func (c *PersistentVirtualClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.current
}

func (c *PersistentVirtualClock) Advance(by time.Duration) error {
	if by <= 0 {
		return ErrInvalidAdvance
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	next := c.current.Add(by)
	if !next.After(c.current) {
		return ErrBackwardAdvance
	}
	if err := c.store.Save(context.Background(), c.key, next); err != nil {
		return err
	}
	c.current = next.UTC()
	return nil
}

func (c *PersistentVirtualClock) Restore(at time.Time) error {
	if at.IsZero() {
		return ErrInvalidTime
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.store.Save(context.Background(), c.key, at); err != nil {
		return err
	}
	c.current = at.UTC()
	return nil
}

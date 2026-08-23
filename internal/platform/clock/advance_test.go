package clock

import (
	"errors"
	"testing"
	"time"
)

func TestTimeAdvancer(t *testing.T) {
	initial := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC)
	virtualClock := mustClock(t, initial)
	advancer, err := NewTimeAdvancer(virtualClock)
	if err != nil {
		t.Fatalf("NewTimeAdvancer() error = %v", err)
	}

	result, err := advancer.Execute(AdvanceTimeCommand{By: 5 * time.Minute})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.Current.Equal(initial.Add(5 * time.Minute)) {
		t.Fatalf("Current = %v, want %v", result.Current, initial.Add(5*time.Minute))
	}
}

func TestTimeAdvancer_RejectsInvalidDuration(t *testing.T) {
	virtualClock := mustClock(t, time.Unix(1, 0))
	advancer, err := NewTimeAdvancer(virtualClock)
	if err != nil {
		t.Fatalf("NewTimeAdvancer() error = %v", err)
	}

	for _, duration := range []time.Duration{0, -time.Second} {
		if _, err := advancer.Execute(AdvanceTimeCommand{By: duration}); !errors.Is(err, ErrInvalidAdvance) {
			t.Fatalf("Execute(%v) error = %v, want %v", duration, err, ErrInvalidAdvance)
		}
	}

	if got := virtualClock.Now(); !got.Equal(time.Unix(1, 0)) {
		t.Fatalf("clock changed after invalid commands: %v", got)
	}
}

func TestTimeAdvancer_PropagatesClockError(t *testing.T) {
	wantErr := errors.New("advance failed")
	fake := &fakeAdvancingClock{advanceErr: wantErr, now: time.Unix(1, 0)}
	advancer, err := NewTimeAdvancer(fake)
	if err != nil {
		t.Fatalf("NewTimeAdvancer() error = %v", err)
	}

	if _, err := advancer.Execute(AdvanceTimeCommand{By: time.Second}); !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}

	if fake.advanceCalls != 1 {
		t.Fatalf("Advance() calls = %d, want 1", fake.advanceCalls)
	}
}

func TestNewTimeAdvancer_RejectsNilClock(t *testing.T) {
	if _, err := NewTimeAdvancer(nil); !errors.Is(err, ErrInvalidClock) {
		t.Fatalf("NewTimeAdvancer() error = %v, want %v", err, ErrInvalidClock)
	}
}

type fakeAdvancingClock struct {
	now          time.Time
	advanceErr   error
	advanceCalls int
}

func (c *fakeAdvancingClock) Advance(time.Duration) error {
	c.advanceCalls++
	return c.advanceErr
}

func (c *fakeAdvancingClock) Now() time.Time {
	return c.now
}

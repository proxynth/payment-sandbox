package clock

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewVirtualClock(t *testing.T) {
	initial := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	clock, err := NewVirtualClock(initial)
	if err != nil {
		t.Fatalf("NewVirtualClock() error = %v", err)
	}

	if !clock.Now().Equal(initial) || clock.Now().Location() != time.UTC {
		t.Fatalf("Now() = %v, want UTC equivalent of %v", clock.Now(), initial)
	}
}

func TestNewVirtualClock_RejectsZeroTime(t *testing.T) {
	if _, err := NewVirtualClock(time.Time{}); !errors.Is(err, ErrInvalidTime) {
		t.Fatalf("NewVirtualClock() error = %v, want %v", err, ErrInvalidTime)
	}
}

func TestVirtualClockAdvance(t *testing.T) {
	initial := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC)
	clock := mustClock(t, initial)

	if err := clock.Advance(5 * time.Minute); err != nil {
		t.Fatalf("Advance() error = %v", err)
	}

	want := initial.Add(5 * time.Minute)
	if !clock.Now().Equal(want) {
		t.Fatalf("Now() = %v, want %v", clock.Now(), want)
	}
}

func TestVirtualClockAdvance_RejectsInvalidMovement(t *testing.T) {
	clock := mustClock(t, time.Unix(100, 0))

	for _, advance := range []time.Duration{0, -time.Second} {
		if err := clock.Advance(advance); !errors.Is(err, ErrInvalidAdvance) {
			t.Fatalf("Advance(%v) error = %v, want %v", advance, err, ErrInvalidAdvance)
		}
	}
}

func TestVirtualClockRestore(t *testing.T) {
	clock := mustClock(t, time.Unix(100, 0))
	restored := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.FixedZone("CEST", 2*60*60))

	if err := clock.Restore(restored); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	if !clock.Now().Equal(restored) || clock.Now().Location() != time.UTC {
		t.Fatalf("Now() = %v, want UTC equivalent of %v", clock.Now(), restored)
	}

	if err := clock.Restore(time.Time{}); !errors.Is(err, ErrInvalidTime) {
		t.Fatalf("Restore() error = %v, want %v", err, ErrInvalidTime)
	}
}

func TestVirtualClock_IsSafeForConcurrentReadsAndAdvances(t *testing.T) {
	clock := mustClock(t, time.Unix(0, 0))
	const advances = 100

	var waitGroup sync.WaitGroup
	waitGroup.Add(advances)
	for range advances {
		go func() {
			defer waitGroup.Done()
			if err := clock.Advance(time.Second); err != nil {
				t.Errorf("Advance() error = %v", err)
			}
			_ = clock.Now()
		}()
	}
	waitGroup.Wait()

	if want := time.Duration(advances) * time.Second; !clock.Now().Equal(time.Unix(0, 0).Add(want)) {
		t.Fatalf("Now() = %v, want %v", clock.Now(), time.Unix(0, 0).Add(want))
	}
}

func mustClock(t *testing.T, initial time.Time) *VirtualClock {
	t.Helper()

	clock, err := NewVirtualClock(initial)
	if err != nil {
		t.Fatalf("NewVirtualClock() error = %v", err)
	}

	return clock
}

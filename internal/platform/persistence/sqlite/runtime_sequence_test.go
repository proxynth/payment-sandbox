package sqlite

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/platform/config"
	"proxynth/payment-sandbox/internal/platform/persistence/sqlite/migrations"
)

func TestNextRuntimeSequenceIsUniqueUnderConcurrency(t *testing.T) {
	db, err := Open(context.Background(), config.DatabaseConfig{Path: filepath.Join(t.TempDir(), "sequence.db"), BusyTimeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrations.Up(db); err != nil {
		t.Fatal(err)
	}

	const workers = 32
	values := make(chan uint64, workers)
	errs := make(chan error, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			value, err := NextRuntimeSequence(context.Background(), db)
			if err != nil {
				errs <- err
				return
			}
			values <- value
		}()
	}
	group.Wait()
	close(values)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}

	seen := make(map[uint64]struct{}, workers)
	for value := range values {
		if _, exists := seen[value]; exists {
			t.Fatalf("runtime sequence %d allocated more than once", value)
		}
		seen[value] = struct{}{}
	}
	if len(seen) != workers {
		t.Fatalf("allocated sequences = %d, want %d", len(seen), workers)
	}
}

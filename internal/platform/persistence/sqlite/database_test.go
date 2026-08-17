package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/platform/config"
)

func TestOpen_CreatesDatabase(t *testing.T) {
	ctx := context.Background()

	path := filepath.Join(t.TempDir(), "payment-sandbox.db")

	db, err := Open(ctx, config.DatabaseConfig{
		Path:        path,
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			t.Fatalf("Close() error: %v", err)
		}
	}(db)

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("db.PingContext() error: %v", err)
	}
}

func TestOpen_ConfiguresNumericPragmas(t *testing.T) {
	ctx := context.Background()

	dbPath := filepath.Join(t.TempDir(), "payment-sandbox.db")

	db, err := Open(ctx, config.DatabaseConfig{
		Path:        dbPath,
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	tests := []struct {
		name  string
		query string
		want  int
	}{
		{
			name:  "foreign keys enabled",
			query: "PRAGMA foreign_keys;",
			want:  1,
		},
		{
			name:  "synchronous mode is NORMAL",
			query: "PRAGMA synchronous;",
			want:  1,
		},
		{
			name:  "busy timeout configured",
			query: "PRAGMA busy_timeout;",
			want:  5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got int

			if err := db.QueryRowContext(ctx, tt.query).Scan(&got); err != nil {
				t.Fatalf("query %q: %v", tt.query, err)
			}

			if got != tt.want {
				t.Errorf("%s = %d, want %d", tt.query, got, tt.want)
			}
		})
	}
}

func TestOpen_ConfiguresWALMode(t *testing.T) {
	ctx := context.Background()

	dbPath := filepath.Join(t.TempDir(), "payment-sandbox.db")

	db, err := Open(ctx, config.DatabaseConfig{
		Path:        dbPath,
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	var got string

	if err := db.QueryRowContext(
		ctx,
		"PRAGMA journal_mode;",
	).Scan(&got); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}

	if got != "wal" {
		t.Errorf("journal_mode = %q, want %q", got, "wal")
	}
}

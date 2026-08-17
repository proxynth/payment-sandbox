package migrations

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/platform/config"
	"proxynth/payment-sandbox/internal/platform/persistence/sqlite"

	"github.com/pressly/goose/v3"
)

const latestVersion int64 = 2

func TestUp_AppliesMigrations(t *testing.T) {
	db := openTestDatabase(t)

	if err := Up(db); err != nil {
		t.Fatalf("Up() error = %v", err)
	}
}

func TestUp_CanRunMultipleTimes(t *testing.T) {
	db := openTestDatabase(t)

	if err := Up(db); err != nil {
		t.Fatalf("first Up() error = %v", err)
	}

	if err := Up(db); err != nil {
		t.Fatalf("second Up() error = %v", err)
	}
}

func TestUp_TracksCurrentVersion(t *testing.T) {
	db := openTestDatabase(t)

	if err := Up(db); err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	version, err := goose.GetDBVersion(db)
	if err != nil {
		t.Fatalf("GetDBVersion() error = %v", err)
	}

	if version != latestVersion {
		t.Errorf("database version = %d, want %d", version, latestVersion)
	}
}

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	ctx := context.Background()

	path := filepath.Join(t.TempDir(), "payment-sandbox.db")

	db, err := sqlite.Open(ctx, config.DatabaseConfig{
		Path:        path,
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("sqlite.Open() error = %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

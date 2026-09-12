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

const latestVersion int64 = 14

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

func TestUp_VersionFourteenPreservesPopulatedVersionThirteen(t *testing.T) {
	db := openTestDatabase(t)
	goose.SetBaseFS(files)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(db, "sql", 13); err != nil {
		t.Fatalf("migrate to version 13: %v", err)
	}

	ctx := context.Background()
	statements := []string{
		`INSERT INTO event_log(id,aggregate_id,event_type,occurred_at,aggregate_version,correlation_id,causation_id,payload) VALUES ('event-existing','payment-existing','payment.created','2026-01-01T00:00:00.000000000Z',1,'correlation','',X'7B7D')`,
		`INSERT INTO scheduler_job_audit(id,job_id,job_type,payload,status,attempts,scheduled_at,next_attempt_at,lease_owner,lease_expires_at,aggregate_id,causation_id) VALUES ('audit-existing','job-existing','webhook.delivery',X'7B7D','pending',0,'2026-01-01T00:00:00.000000000Z','2026-01-01T00:00:00.000000000Z','','','payment-existing','event-existing')`,
		`INSERT INTO webhook_delivery_audit(job_id,attempt,endpoint_id,correlation_id,causation_id,outcome,http_status,error) VALUES ('job-existing',1,'endpoint-existing','correlation','event-existing','started',0,'')`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("seed version 13 data: %v", err)
		}
	}

	if err := Up(db); err != nil {
		t.Fatalf("upgrade populated version 13 database: %v", err)
	}

	checks := []struct {
		name  string
		query string
	}{
		{"event", `SELECT runtime_sequence FROM event_log WHERE id='event-existing'`},
		{"job snapshot", `SELECT runtime_sequence FROM scheduler_job_audit WHERE id='audit-existing'`},
		{"webhook attempt", `SELECT runtime_sequence FROM webhook_delivery_audit WHERE job_id='job-existing' AND attempt=1`},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			var sequence uint64
			if err := db.QueryRowContext(ctx, check.query).Scan(&sequence); err != nil {
				t.Fatalf("read migrated runtime sequence: %v", err)
			}
			if sequence != 0 {
				t.Fatalf("migrated runtime sequence = %d, want 0 for legacy record", sequence)
			}
		})
	}

	var value uint64
	if err := db.QueryRowContext(ctx, `SELECT value FROM runtime_sequence WHERE id=1`).Scan(&value); err != nil {
		t.Fatalf("read sequence allocator: %v", err)
	}
	if value != 0 {
		t.Fatalf("initial allocator value = %d, want 0", value)
	}
	var integrity string
	if err := db.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil {
		t.Fatalf("PRAGMA integrity_check: %v", err)
	}
	if integrity != "ok" {
		t.Fatalf("PRAGMA integrity_check = %q, want ok", integrity)
	}
}

func TestUp_VersionFourteenFailureRollsBackPartialSchemaChanges(t *testing.T) {
	db := openTestDatabase(t)
	goose.SetBaseFS(files)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(db, "sql", 13); err != nil {
		t.Fatalf("migrate to version 13: %v", err)
	}

	// Force migration 14 to fail after creating and seeding runtime_sequence.
	if _, err := db.ExecContext(context.Background(), `ALTER TABLE event_log ADD COLUMN runtime_sequence INTEGER NOT NULL DEFAULT 0`); err != nil {
		t.Fatalf("prepare conflicting schema: %v", err)
	}
	if err := Up(db); err == nil {
		t.Fatal("Up() error = nil, want duplicate-column migration failure")
	}

	var tableCount int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='runtime_sequence'`).Scan(&tableCount); err != nil {
		t.Fatalf("inspect runtime_sequence table: %v", err)
	}
	if tableCount != 0 {
		t.Fatalf("runtime_sequence table count after failed migration = %d, want 0", tableCount)
	}
	version, err := goose.GetDBVersion(db)
	if err != nil {
		t.Fatalf("read migration version: %v", err)
	}
	if version != 13 {
		t.Fatalf("migration version after failed upgrade = %d, want 13", version)
	}
	var integrity string
	if err := db.QueryRowContext(context.Background(), `PRAGMA integrity_check`).Scan(&integrity); err != nil {
		t.Fatalf("PRAGMA integrity_check: %v", err)
	}
	if integrity != "ok" {
		t.Fatalf("PRAGMA integrity_check = %q, want ok", integrity)
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

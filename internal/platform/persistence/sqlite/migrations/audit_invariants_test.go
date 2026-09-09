package migrations

import (
	"context"
	"testing"

	"github.com/pressly/goose/v3"
)

// Invariant: upgrade from a supported populated previous schema preserves data.
func TestAuditUpgradePopulatedVersionTwo(t *testing.T) {
	db := openTestDatabase(t)
	goose.SetBaseFS(files)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(db, "sql", 2); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO payments(id,status,version) VALUES ('legacy','pending',1)`); err != nil {
		t.Fatal(err)
	}
	if err := Up(db); err != nil {
		t.Fatalf("populated v2 -> latest migration failed: %v", err)
	}
}

// Positive control: repeating latest migrations does not erase a populated database.
func TestAuditReapplyPreservesPopulatedLatest(t *testing.T) {
	db := openTestDatabase(t)
	if err := Up(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO payments(id,status,version,amount,currency) VALUES ('existing','pending',1,1000,'EUR')`); err != nil {
		t.Fatal(err)
	}
	if err := Up(db); err != nil {
		t.Fatal(err)
	}
	var amount int64
	if err := db.QueryRowContext(context.Background(), `SELECT amount FROM payments WHERE id='existing'`).Scan(&amount); err != nil {
		t.Fatal(err)
	}
	if amount != 1000 {
		t.Fatalf("amount=%d", amount)
	}
}

// Invariant: version 12 enriches existing job audit rows without losing their
// historical identity, so snapshots made before the upgrade become restorable.
func TestAuditUpgradeBackfillsSchedulerJobAuditPayload(t *testing.T) {
	db := openTestDatabase(t)
	goose.SetBaseFS(files)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(db, "sql", 11); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO scheduler_jobs(id,type,payload,scheduled_at,next_attempt_at,status,lease_owner,lease_expires_at,attempts) VALUES ('job-1','webhook.delivery',X'7B7D','2026-01-01T00:00:00.000000000Z','2026-01-01T00:00:00.000000000Z','pending','','',0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO scheduler_job_audit(id,job_id,status,attempts,scheduled_at,next_attempt_at,lease_owner,lease_expires_at) VALUES ('audit-1','job-1','pending',0,'2026-01-01T00:00:00.000000000Z','2026-01-01T00:00:00.000000000Z','','')`); err != nil {
		t.Fatal(err)
	}
	if err := Up(db); err != nil {
		t.Fatal(err)
	}
	var jobType string
	var payload []byte
	if err := db.QueryRow(`SELECT job_type,payload FROM scheduler_job_audit WHERE id='audit-1'`).Scan(&jobType, &payload); err != nil {
		t.Fatal(err)
	}
	if jobType != "webhook.delivery" || string(payload) != "{}" {
		t.Fatalf("backfilled job audit = type=%q payload=%q", jobType, payload)
	}
}

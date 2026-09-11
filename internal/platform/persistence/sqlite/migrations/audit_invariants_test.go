package migrations

import (
	"context"
	"testing"

	sqlite "proxynth/payment-sandbox/internal/platform/persistence/sqlite"

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
	if _, err := db.ExecContext(context.Background(), `INSERT INTO scheduler_jobs(id,type,payload,scheduled_at,next_attempt_at,status,lease_owner,lease_expires_at,attempts) VALUES ('job-1','webhook.delivery',X'7B7D','2026-01-01T00:00:00.000000000Z','2026-01-01T00:00:00.000000000Z','pending','','',0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO scheduler_job_audit(id,job_id,status,attempts,scheduled_at,next_attempt_at,lease_owner,lease_expires_at) VALUES ('audit-1','job-1','pending',0,'2026-01-01T00:00:00.000000000Z','2026-01-01T00:00:00.000000000Z','','')`); err != nil {
		t.Fatal(err)
	}
	if err := Up(db); err != nil {
		t.Fatal(err)
	}
	var jobType string
	var payload []byte
	if err := db.QueryRowContext(context.Background(), `SELECT job_type,payload FROM scheduler_job_audit WHERE id='audit-1'`).Scan(&jobType, &payload); err != nil {
		t.Fatal(err)
	}
	if jobType != "webhook.delivery" || string(payload) != "{}" {
		t.Fatalf("backfilled job audit = type=%q payload=%q", jobType, payload)
	}
}

// Invariant: version 12 keeps an audit snapshot when its original job was
// already deleted, while making the loss of its payload explicit through the
// migration defaults.
func TestAuditUpgradeKeepsSnapshotWhenOriginalJobIsMissing(t *testing.T) {
	db := openTestDatabase(t)
	goose.SetBaseFS(files)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(db, "sql", 11); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO scheduler_job_audit(id,job_id,status,attempts,scheduled_at,next_attempt_at,lease_owner,lease_expires_at) VALUES ('audit-missing','deleted-job','completed',1,'2026-01-01T00:00:00.000000000Z','2026-01-01T00:00:00.000000000Z','','')`); err != nil {
		t.Fatal(err)
	}
	if err := Up(db); err != nil {
		t.Fatal(err)
	}
	var jobType string
	var payload []byte
	if err := db.QueryRowContext(context.Background(), `SELECT job_type,payload FROM scheduler_job_audit WHERE id='audit-missing'`).Scan(&jobType, &payload); err != nil {
		t.Fatal(err)
	}
	if jobType != "" || len(payload) != 0 {
		t.Fatalf("missing-job audit = type=%q payload=%q, want explicit empty defaults", jobType, payload)
	}
}

// Invariant: new durable runtime records receive one monotonically increasing
// sequence shared by events, job snapshots and delivery attempts.
func TestAuditRuntimeSequenceOrdersDurableRecords(t *testing.T) {
	db := openTestDatabase(t)
	if err := Up(db); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	sequence, err := sqlite.NextRuntimeSequence(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO event_log(id,aggregate_id,event_type,occurred_at,aggregate_version,correlation_id,causation_id,payload,runtime_sequence) VALUES ('event-sequence','payment-sequence','payment.created','2026-01-01T00:00:00.000000000Z',1,'corr','',X'',$1)`, sequence); err != nil {
		t.Fatal(err)
	}
	sequence, err = sqlite.NextRuntimeSequence(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO scheduler_job_audit(id,job_id,job_type,payload,status,attempts,scheduled_at,next_attempt_at,lease_owner,lease_expires_at,aggregate_id,causation_id,runtime_sequence) VALUES ('audit-sequence','job-sequence','webhook.delivery',X'','pending',0,'2026-01-01T00:00:00.000000000Z','2026-01-01T00:00:00.000000000Z','','','payment-sequence','event-sequence',$1)`, sequence); err != nil {
		t.Fatal(err)
	}
	sequence, err = sqlite.NextRuntimeSequence(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO webhook_delivery_audit(job_id,attempt,endpoint_id,correlation_id,causation_id,outcome,http_status,error,runtime_sequence) VALUES ('job-sequence',1,'endpoint-sequence','corr','event-sequence','started',0,'',$1)`, sequence); err != nil {
		t.Fatal(err)
	}
	rows, err := db.QueryContext(ctx, `SELECT runtime_sequence FROM event_log WHERE id='event-sequence' UNION ALL SELECT runtime_sequence FROM scheduler_job_audit WHERE id='audit-sequence' UNION ALL SELECT runtime_sequence FROM webhook_delivery_audit WHERE job_id='job-sequence' ORDER BY runtime_sequence`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var sequences []uint64
	for rows.Next() {
		var sequence uint64
		if err := rows.Scan(&sequence); err != nil {
			t.Fatal(err)
		}
		sequences = append(sequences, sequence)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(sequences) != 3 || sequences[0] == 0 || sequences[1] != sequences[0]+1 || sequences[2] != sequences[1]+1 {
		t.Fatalf("runtime sequences = %v, want three consecutive values", sequences)
	}
}

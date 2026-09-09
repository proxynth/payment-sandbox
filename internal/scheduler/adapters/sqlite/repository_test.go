package sqlite

import (
	"context"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/platform/config"
	persistencesqlite "proxynth/payment-sandbox/internal/platform/persistence/sqlite"
	"proxynth/payment-sandbox/internal/platform/persistence/sqlite/migrations"
	"proxynth/payment-sandbox/internal/scheduler/domain"
)

func TestRepositoryRoundTripsAndAcquiresJob(t *testing.T) {
	db, err := persistencesqlite.Open(context.Background(), config.DatabaseConfig{Path: t.TempDir() + "/jobs.db", BusyTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := migrations.Up(db); err != nil {
		t.Fatal(err)
	}

	at := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	job, err := domain.NewJob("saga:authorize:1", "saga.step", []byte(`{"step":"authorize"}`), at)
	if err != nil {
		t.Fatal(err)
	}
	repository := NewRepository(db)
	if err := repository.Save(context.Background(), &job); err != nil {
		t.Fatal(err)
	}

	jobs, err := repository.FindExecutable(context.Background(), at, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("jobs = %d, want 1", len(jobs))
	}
	acquired, err := repository.Acquire(context.Background(), jobs[0].ID(), "worker-1", at.Add(time.Minute), at)
	if err != nil {
		t.Fatal(err)
	}
	if acquired.Status() != domain.JobLeased || acquired.LeaseOwner() != "worker-1" {
		t.Fatalf("acquired job = %+v", acquired)
	}
	var snapshots int
	if err := db.QueryRow(`SELECT count(*) FROM scheduler_job_audit WHERE job_id = ?`, job.ID()).Scan(&snapshots); err != nil {
		t.Fatal(err)
	}
	if snapshots != 2 {
		t.Fatalf("audit snapshots = %d, want 2", snapshots)
	}
}

// Invariant: a lifecycle snapshot contains enough immutable data to restore
// the dispatched job after the mutable scheduler_jobs row is unavailable.
func TestRepositoryAuditSnapshotsRestoreCompletedJob(t *testing.T) {
	db, err := persistencesqlite.Open(context.Background(), config.DatabaseConfig{Path: t.TempDir() + "/job-audit.db", BusyTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := migrations.Up(db); err != nil {
		t.Fatal(err)
	}

	at := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	job, err := domain.NewJob("job-audit", "webhook.delivery", []byte(`{"event":"payment.authorized"}`), at, domain.JobMetadata{AggregateID: "payment-1", CausationID: "event-1"})
	if err != nil {
		t.Fatal(err)
	}
	repository := NewRepository(db)
	if err := repository.Save(context.Background(), &job); err != nil {
		t.Fatal(err)
	}
	if err := job.Lease("worker-1", at.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(context.Background(), &job); err != nil {
		t.Fatal(err)
	}
	if err := job.Start(); err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(context.Background(), &job); err != nil {
		t.Fatal(err)
	}
	if err := job.Complete(); err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(context.Background(), &job); err != nil {
		t.Fatal(err)
	}

	snapshots, err := repository.ListAudit(context.Background(), job.ID())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 4 {
		t.Fatalf("snapshots=%d, want 4", len(snapshots))
	}
	restored, err := snapshots[len(snapshots)-1].Restore()
	if err != nil {
		t.Fatal(err)
	}
	if restored.ID() != job.ID() || restored.Type() != job.Type() || string(restored.Payload()) != string(job.Payload()) || restored.AggregateID() != "payment-1" || restored.CausationID() != "event-1" || restored.Status() != domain.JobCompleted || restored.Attempts() != 1 {
		t.Fatalf("restored job = %#v", restored)
	}
}

// Invariant: aggregate history releases its cursor before reading individual
// job histories on the runtime's single-connection SQLite pool.
func TestAggregateHistoryWithSingleConnection(t *testing.T) {
	db, err := persistencesqlite.Open(context.Background(), config.DatabaseConfig{Path: t.TempDir() + "/history.db", BusyTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Up(db); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)
	job, err := domain.NewJob("history-job", "saga.step", []byte(`{}`), time.Unix(1, 0), domain.JobMetadata{AggregateID: "payment-1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(context.Background(), &job); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	snapshots, err := repo.ListAuditByAggregate(ctx, "payment-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 1 || snapshots[0].ID != job.ID() {
		t.Fatalf("snapshots=%+v", snapshots)
	}
}

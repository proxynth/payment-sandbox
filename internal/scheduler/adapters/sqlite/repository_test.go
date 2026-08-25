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
	acquired, err := repository.Acquire(context.Background(), jobs[0].ID(), "worker-1", at.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if acquired.Status() != domain.JobLeased || acquired.LeaseOwner() != "worker-1" {
		t.Fatalf("acquired job = %+v", acquired)
	}
}

package sqlite

import (
	"context"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/platform/config"
	persistencesqlite "proxynth/payment-sandbox/internal/platform/persistence/sqlite"
	"proxynth/payment-sandbox/internal/platform/persistence/sqlite/migrations"
	"proxynth/payment-sandbox/internal/webhook/application"
)

// Invariant: a result is retained once per durable scheduler attempt, even if
// failure handling re-enters the persistence adapter.
func TestDeliveryAuditRepositoryPersistsOneRecordPerJobAttempt(t *testing.T) {
	db, err := persistencesqlite.Open(context.Background(), config.DatabaseConfig{Path: t.TempDir() + "/webhook-audit.db", BusyTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := migrations.Up(db); err != nil {
		t.Fatal(err)
	}

	repository := NewDeliveryAuditRepository(db)
	started := application.DeliveryAttempt{
		JobID: "job-1", Attempt: 1, EndpointID: "endpoint-1", CorrelationID: "request-1", CausationID: "event-1",
		Outcome: application.DeliveryStarted,
	}
	if err := repository.Record(context.Background(), started); err != nil {
		t.Fatal(err)
	}
	failed := started
	failed.Outcome = application.DeliveryFailed
	failed.HTTPStatus = 502
	failed.Error = "callback delivery failed: unexpected HTTP status 502"
	if err := repository.Record(context.Background(), failed); err != nil {
		t.Fatal(err)
	}
	if err := repository.Record(context.Background(), started); err != nil {
		t.Fatal(err)
	}

	var attempts, status int
	var outcome, recordedErr string
	if err := db.QueryRowContext(context.Background(), `SELECT attempt, http_status, outcome, error FROM webhook_delivery_audit WHERE job_id = ?`, "job-1").Scan(&attempts, &status, &outcome, &recordedErr); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 || status != 502 || outcome != string(application.DeliveryFailed) || recordedErr != failed.Error {
		t.Fatalf("persisted audit = attempt=%d status=%d outcome=%q error=%q", attempts, status, outcome, recordedErr)
	}
}

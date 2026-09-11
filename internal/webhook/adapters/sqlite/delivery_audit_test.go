package sqlite

import (
	"context"
	"sync"
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
	startedMismatch := started
	startedMismatch.CausationID = "different-event"
	if err := repository.Record(context.Background(), startedMismatch); err == nil {
		t.Fatal("started marker with contradictory metadata was accepted")
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

func TestDeliveryAuditRepositoryDoesNotOverwriteTerminalResult(t *testing.T) {
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
	succeeded := started
	succeeded.Outcome = application.DeliverySucceeded
	succeeded.HTTPStatus = 200
	if err := repository.Record(context.Background(), succeeded); err != nil {
		t.Fatal(err)
	}
	if err := repository.Record(context.Background(), succeeded); err != nil {
		t.Fatalf("repeating the same terminal result should be idempotent: %v", err)
	}
	metadataMismatch := succeeded
	metadataMismatch.CorrelationID = "different-request"
	if err := repository.Record(context.Background(), metadataMismatch); err == nil {
		t.Fatal("terminal result with contradictory correlation metadata was accepted")
	}
	contradictory := succeeded
	contradictory.Outcome = application.DeliveryFailed
	contradictory.HTTPStatus = 502
	contradictory.Error = "callback delivery failed: unexpected HTTP status 502"
	if err := repository.Record(context.Background(), contradictory); err == nil {
		t.Fatal("contradictory terminal result was accepted")
	}

	var status int
	var outcome, recordedErr string
	if err := db.QueryRowContext(context.Background(), `SELECT http_status, outcome, error FROM webhook_delivery_audit WHERE job_id = ? AND attempt = ?`, "job-1", 1).Scan(&status, &outcome, &recordedErr); err != nil {
		t.Fatal(err)
	}
	if status != 200 || outcome != string(application.DeliverySucceeded) || recordedErr != "" {
		t.Fatalf("persisted audit = status=%d outcome=%q error=%q", status, outcome, recordedErr)
	}
}

func TestDeliveryAuditRepositorySerializesConcurrentTerminalResults(t *testing.T) {
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

	succeeded := started
	succeeded.Outcome = application.DeliverySucceeded
	succeeded.HTTPStatus = 200
	failed := started
	failed.Outcome = application.DeliveryFailed
	failed.HTTPStatus = 502
	failed.Error = "callback delivery failed: unexpected HTTP status 502"

	start := make(chan struct{})
	errs := make(chan error, 2)
	var group sync.WaitGroup
	group.Add(2)
	go func() {
		defer group.Done()
		<-start
		errs <- repository.Record(context.Background(), succeeded)
	}()
	go func() {
		defer group.Done()
		<-start
		errs <- repository.Record(context.Background(), failed)
	}()
	close(start)
	group.Wait()
	close(errs)

	var successes, failures int
	for err := range errs {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("concurrent terminal results = successes=%d failures=%d", successes, failures)
	}

	var count int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM webhook_delivery_audit WHERE job_id = ? AND attempt = ?`, "job-1", 1).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("audit record count = %d, want 1", count)
	}
}

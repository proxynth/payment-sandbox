package application_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	admin "proxynth/payment-sandbox/internal/administration/application"
	paymentsqlite "proxynth/payment-sandbox/internal/payment/adapters/sqlite"
	payment "proxynth/payment-sandbox/internal/payment/domain"
	"proxynth/payment-sandbox/internal/platform/config"
	persistence "proxynth/payment-sandbox/internal/platform/persistence/sqlite"
	"proxynth/payment-sandbox/internal/platform/persistence/sqlite/migrations"
	jobsqlite "proxynth/payment-sandbox/internal/scheduler/adapters/sqlite"
	jobs "proxynth/payment-sandbox/internal/scheduler/domain"
	hooksqlite "proxynth/payment-sandbox/internal/webhook/adapters/sqlite"
	hooks "proxynth/payment-sandbox/internal/webhook/application"
)

type pausedEvents struct {
	*paymentsqlite.EventLogRepository
	afterRead func() error
}

func (r *pausedEvents) ListByAggregate(ctx context.Context, id payment.ID) ([]payment.BusinessEvent, error) {
	events, err := r.EventLogRepository.ListByAggregate(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := r.afterRead(); err != nil {
		return nil, err
	}
	return events, nil
}

// Invariant: an audit response sees one committed SQLite snapshot even when
// a separate connection commits an event, job and delivery between its reads.
func TestRuntimeHistoryUsesOneSQLiteSnapshot(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cfg := config.DatabaseConfig{Path: t.TempDir() + "/history.db", BusyTimeout: time.Second}
	db, err := persistence.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := migrations.Up(db); err != nil {
		t.Fatal(err)
	}
	writer, err := persistence.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Close() }()
	money, err := payment.NewMoney(1000, "EUR")
	if err != nil {
		t.Fatal(err)
	}
	p, err := payment.New("snapshot-payment", money)
	if err != nil {
		t.Fatal(err)
	}
	event := func(id payment.EventID, kind payment.EventType) payment.BusinessEvent {
		payload, err := json.Marshal(payment.NewEventSnapshot(p))
		if err != nil {
			t.Fatal(err)
		}
		e, err := payment.NewBusinessEventWithPayload(id, p.ID(), kind, time.Unix(int64(p.Version()), 0), p.Version(), "corr", "", payload)
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
	events := paymentsqlite.NewEventLogRepository(db)
	if err := events.Append(ctx, event("created", payment.EventPaymentCreated)); err != nil {
		t.Fatal(err)
	}
	if err := p.Authorize(); err != nil {
		t.Fatal(err)
	}
	authorized := event("authorized", payment.EventPaymentAuthorized)
	job, err := jobs.NewJob("new-job", "webhook.delivery", []byte(`{}`), time.Unix(2, 0), jobs.JobMetadata{AggregateID: string(p.ID()), CausationID: "authorized"})
	if err != nil {
		t.Fatal(err)
	}
	otherJob, err := jobs.NewJob("other-job", "webhook.delivery", []byte(`{"other":true}`), time.Unix(2, 0), jobs.JobMetadata{AggregateID: "other-payment", CausationID: "other-event"})
	if err != nil {
		t.Fatal(err)
	}
	if err := jobsqlite.NewRepository(db).Save(ctx, &otherJob); err != nil {
		t.Fatal(err)
	}
	write := func() error {
		return persistence.NewTransactionManager(writer).WithinContext(ctx, func(txctx context.Context) error {
			if err := paymentsqlite.NewEventLogRepository(writer).Append(txctx, authorized); err != nil {
				return err
			}
			if err := jobsqlite.NewRepository(writer).Save(txctx, &job); err != nil {
				return err
			}
			return hooksqlite.NewDeliveryAuditRepository(writer).Record(txctx, hooks.DeliveryAttempt{JobID: "new-job", Attempt: 1, EndpointID: "endpoint", Outcome: hooks.DeliveryStarted})
		})
	}
	read := &pausedEvents{EventLogRepository: events, afterRead: func() error {
		done := make(chan error, 1)
		go func() { done <- write() }()
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}}
	history, err := admin.NewRuntimeHistory(persistence.NewTransactionManager(db), read, jobsqlite.NewRepository(db), hooksqlite.NewDeliveryAuditRepository(db))
	if err != nil {
		t.Fatal(err)
	}
	result, err := history.Execute(ctx, p.ID())
	if err != nil {
		t.Fatal(err)
	}
	if result.Payment.Version() != 1 || len(result.Events) != 1 || len(result.Jobs) != 0 {
		t.Fatalf("mixed snapshot: payment version=%d events=%d jobs=%d", result.Payment.Version(), len(result.Events), len(result.Jobs))
	}
	read.afterRead = func() error { return nil }
	result, err = history.Execute(ctx, p.ID())
	if err != nil {
		t.Fatal(err)
	}
	if result.Payment.Version() != 2 || len(result.Events) != 2 || len(result.Jobs) != 1 || len(result.Jobs[0].Deliveries) != 1 {
		t.Fatalf("next read did not see complete committed update: %+v", result)
	}
}

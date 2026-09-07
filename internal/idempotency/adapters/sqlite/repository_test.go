package sqlite

import (
	"context"
	"errors"
	"testing"

	"proxynth/payment-sandbox/internal/idempotency/application"
	"proxynth/payment-sandbox/internal/platform/config"
	sqlite "proxynth/payment-sandbox/internal/platform/persistence/sqlite"
	"proxynth/payment-sandbox/internal/platform/persistence/sqlite/migrations"
)

func TestRepositoryReserveAndComplete(t *testing.T) {
	db, err := sqlite.Open(context.Background(), config.DatabaseConfig{Path: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := migrations.Up(db); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)
	record := application.Record{Scope: "payment:p", Key: "k", Fingerprint: "f", Status: "processing"}
	reserved, err := repo.Reserve(context.Background(), record)
	if err != nil || !reserved {
		t.Fatalf("reserve=%v err=%v", reserved, err)
	}
	reserved, err = repo.Reserve(context.Background(), record)
	if err != nil || reserved {
		t.Fatalf("duplicate reserve=%v err=%v", reserved, err)
	}
	record.Status, record.ResponseStatus, record.ResponseBody = "completed", 200, []byte(`{"ok":true}`)
	if err := repo.Complete(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Find(context.Background(), record.Scope, record.Key)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "completed" || string(got.ResponseBody) != string(record.ResponseBody) {
		t.Fatalf("got=%+v", got)
	}
	_, err = repo.Reserve(context.Background(), application.Record{Scope: record.Scope, Key: record.Key, Fingerprint: "different", Status: "processing"})
	if !errors.Is(err, application.ErrFingerprintConflict) {
		t.Fatalf("err=%v", err)
	}
}

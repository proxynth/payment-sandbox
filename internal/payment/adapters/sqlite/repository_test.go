package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/payment/application"
	"proxynth/payment-sandbox/internal/payment/domain"
	"proxynth/payment-sandbox/internal/platform/config"
	platformsqlite "proxynth/payment-sandbox/internal/platform/persistence/sqlite"
	"proxynth/payment-sandbox/internal/platform/persistence/sqlite/migrations"
)

func newTestRepository(t *testing.T) *Repository {
	t.Helper()

	return NewRepository(newTestDatabase(t))
}

func newTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	ctx := context.Background()

	path := filepath.Join(t.TempDir(), "payment-sandbox.db")

	db, err := platformsqlite.Open(ctx, config.DatabaseConfig{
		Path:        path,
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("opening sqlite db: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := migrations.Up(db); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	return db
}

func TestRepository_SaveAndFindByID(t *testing.T) {
	ctx := context.Background()
	repository := newTestRepository(t)

	payment, err := domain.New("test-payment-01")
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	if err := repository.Save(ctx, payment); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	got, err := repository.FindByID(ctx, payment.ID())
	if err != nil {
		t.Fatalf("FindByID() error: %v", err)
	}

	if got.ID() != payment.ID() {
		t.Errorf("ID() = %q, want %q", got.ID(), payment.ID())
	}

	if got.Status() != payment.Status() {
		t.Errorf("Status() = %q, want %q", got.Status(), payment.Status())
	}

	if got.Version() != payment.Version() {
		t.Errorf("Version() = %q, want %q", got.Version(), payment.Version())
	}
}

func TestRepository_SaveSamePaymentTwiceKeepsSingleRow(t *testing.T) {
	ctx := context.Background()
	repository := newTestRepository(t)

	payment, err := domain.New("test-payment-02")
	if err != nil {
		t.Fatalf("domain.New() error = %v", err)
	}

	if err := repository.Save(context.Background(), payment); err != nil {
		t.Fatalf("first Save() error: %v", err)
	}

	if err := repository.Save(context.Background(), payment); err != nil {
		t.Fatalf("second Save() error: %v", err)
	}

	var count int

	err = repository.db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM payments WHERE id = ?",
		payment.ID(),
	).Scan(&count)
	if err != nil {
		t.Fatalf("count persisted payments: %v", err)
	}

	if count != 1 {
		t.Errorf("payment row count = %d, want 1", count)
	}
}

func TestRepository_FindByIDReturnsNotFoundForMissingPayment(t *testing.T) {
	ctx := context.Background()
	repository := newTestRepository(t)

	_, err := repository.FindByID(ctx, "test-payment-missing")

	if !errors.Is(err, application.ErrPaymentNotFound) {
		t.Errorf(
			"FindByID() error = %v, want %v",
			err,
			application.ErrPaymentNotFound,
		)
	}
}

func TestRepository_FindByIDRestoresAggregate(t *testing.T) {
	ctx := context.Background()
	repository := newTestRepository(t)

	payment, err := domain.New("test-payment-04")
	if err != nil {
		t.Fatalf("domain.New() error = %v", err)
	}

	if err := repository.Save(ctx, payment); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	got, err := repository.FindByID(ctx, payment.ID())
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if got.ID() != payment.ID() {
		t.Errorf("ID() = %q, want %q", got.ID(), payment.ID())
	}

	if got.Status() != domain.StatusPending {
		t.Errorf("Status() = %q, want %q", got.Status(), domain.StatusPending)
	}

	if got.Version() != payment.Version() {
		t.Errorf("Version() = %q, want %q", got.Version(), payment.Version())
	}
}

func TestRepository_FindByIDRejectsInvalidPersistedAggregate(t *testing.T) {
	ctx := context.Background()
	repository := newTestRepository(t)

	_, err := repository.db.ExecContext(
		ctx,
		`
		INSERT INTO payments (
		                      id,
		                      status,
		                      version
			  ) VALUES (?, ?, ?)
			  `,
		"test-payment-invalid",
		"foobar",
		1,
	)

	if err != nil {
		t.Fatalf("insert invalid payment: %v", err)
	}

	_, err = repository.FindByID(ctx, "test-payment-invalid")

	if !errors.Is(err, domain.ErrInvalidPaymentStatus) {
		t.Errorf("FindByID() error = %v, want %v", err, domain.ErrInvalidPaymentStatus)
	}
}

func TestRepository_CanUseTransaction(t *testing.T) {
	ctx := context.Background()

	db := newTestDatabase(t)

	payment, err := domain.New("payment-transaction")
	if err != nil {
		t.Fatalf("domain.New() error = %v", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	txRepository := NewRepository(tx)

	if err := txRepository.Save(ctx, payment); err != nil {
		_ = tx.Rollback()
		t.Fatalf("Save() error = %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	repository := NewRepository(db)

	got, err := repository.FindByID(ctx, payment.ID())
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if got.ID() != payment.ID() {
		t.Errorf("ID() = %q, want %q", got.ID(), payment.ID())
	}
}

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

func newTestAmount(t *testing.T, amount int64, currency domain.Currency) domain.Money {
	t.Helper()

	moneyAmount, err := domain.NewMoney(amount, currency)

	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}

	return moneyAmount
}

func TestRepository_SaveAndFindByID(t *testing.T) {
	ctx := context.Background()
	repository := newTestRepository(t)

	payment, err := domain.New("test-payment-01", newTestAmount(t, 4999, "EUR"))
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

	payment, err := domain.New("payment-conflict", newTestAmount(t, 4999, "EUR"))
	if err != nil {
		t.Fatalf("domain.New() error = %v", err)
	}

	if err := repository.Save(ctx, payment); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}

	err = repository.Save(ctx, payment)

	if !errors.Is(err, application.ErrPaymentVersionConflict) {
		t.Errorf(
			"second Save() error = %v, want %v",
			err,
			application.ErrPaymentVersionConflict,
		)
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

	payment, err := domain.New("test-payment-04", newTestAmount(t, 4999, "EUR"))
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
		                      amount,
		                      currency,
		                      version
			  ) VALUES (?, ?, ?, ?, ?)
			  `,
		"test-payment-invalid",
		"foobar",
		1200,
		"EUR",
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

	payment, err := domain.New("payment-transaction", newTestAmount(t, 4999, "EUR"))
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

func TestRepository_SaveNextVersionSucceeds(t *testing.T) {
	ctx := context.Background()
	repository := newTestRepository(t)

	payment, err := domain.New("payment-versioned", newTestAmount(t, 4999, "EUR"))
	if err != nil {
		t.Fatalf("domain.New() error = %v", err)
	}

	if err := repository.Save(ctx, payment); err != nil {
		t.Fatalf("Save(version 1) error = %v", err)
	}

	updatedPayment, err := domain.Restore(domain.PaymentState{
		ID:               payment.ID(),
		Amount:           payment.Amount(),
		Status:           payment.Status(),
		AuthorizedAmount: payment.AuthorizedAmount().Amount(),
		CapturedAmount:   payment.CapturedAmount().Amount(),
		RefundedAmount:   payment.RefundedAmount().Amount(),
		Version:          2,
	})
	if err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	if err := repository.Save(ctx, updatedPayment); err != nil {
		t.Fatalf("Save(version 2) error = %v", err)
	}

	got, err := repository.FindByID(ctx, updatedPayment.ID())
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if got.Version() != 2 {
		t.Errorf("Version() = %d, want %d", got.Version(), 2)
	}
}

func TestRepository_SaveRejectsStaleAggregate(t *testing.T) {
	ctx := context.Background()
	repository := newTestRepository(t)

	payment, err := domain.New("payment-concurrent", newTestAmount(t, 4999, "EUR"))
	if err != nil {
		t.Fatalf("domain.New() error = %v", err)
	}

	if err := repository.Save(ctx, payment); err != nil {
		t.Fatalf("initial Save() error = %v", err)
	}

	first, err := repository.FindByID(ctx, payment.ID())
	if err != nil {
		t.Fatalf("first FindByID() error = %v", err)
	}

	second, err := repository.FindByID(ctx, payment.ID())
	if err != nil {
		t.Fatalf("second FindByID() error = %v", err)
	}

	firstUpdated, err := domain.Restore(domain.PaymentState{
		ID:               first.ID(),
		Amount:           first.Amount(),
		Status:           first.Status(),
		AuthorizedAmount: first.AuthorizedAmount().Amount(),
		CapturedAmount:   first.CapturedAmount().Amount(),
		RefundedAmount:   first.RefundedAmount().Amount(),
		Version:          first.Version() + 1,
	})
	if err != nil {
		t.Fatalf("Restore(first) error = %v", err)
	}

	secondUpdated, err := domain.Restore(domain.PaymentState{
		ID:               second.ID(),
		Amount:           second.Amount(),
		Status:           second.Status(),
		AuthorizedAmount: second.AuthorizedAmount().Amount(),
		CapturedAmount:   second.CapturedAmount().Amount(),
		RefundedAmount:   second.RefundedAmount().Amount(),
		Version:          second.Version() + 1,
	})
	if err != nil {
		t.Fatalf("Restore(second) error = %v", err)
	}
	if err := repository.Save(ctx, firstUpdated); err != nil {
		t.Fatalf("Save(first) error = %v", err)
	}

	err = repository.Save(ctx, secondUpdated)

	if !errors.Is(err, application.ErrPaymentVersionConflict) {
		t.Errorf(
			"Save(second) error = %v, want %v",
			err,
			application.ErrPaymentVersionConflict,
		)
	}

	persisted, err := repository.FindByID(ctx, payment.ID())
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if persisted.Version() != 2 {
		t.Errorf(
			"Version() = %d, want 2",
			persisted.Version(),
		)
	}
}

func TestRepository_SavePersistsAggregateStateChanges(t *testing.T) {
	ctx := context.Background()
	repository := newTestRepository(t)

	payment, err := domain.New(
		"payment-authorized",
		newTestAmount(t, 10000, "EUR"),
	)
	if err != nil {
		t.Fatalf("domain.New() error = %v", err)
	}

	if err := repository.Save(ctx, payment); err != nil {
		t.Fatalf("initial Save() error = %v", err)
	}

	if err := payment.Authorize(); err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}

	if err := repository.Save(ctx, payment); err != nil {
		t.Fatalf("updated Save() error = %v", err)
	}

	got, err := repository.FindByID(ctx, payment.ID())
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if got.Status() != domain.StatusAuthorized {
		t.Errorf(
			"Status() = %q, want %q",
			got.Status(),
			domain.StatusAuthorized,
		)
	}

	if got.AuthorizedAmount().Amount() != 10000 {
		t.Errorf(
			"AuthorizedAmount() = %d, want 10000",
			got.AuthorizedAmount().Amount(),
		)
	}
}

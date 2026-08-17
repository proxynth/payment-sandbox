package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/platform/config"
	"proxynth/payment-sandbox/internal/platform/persistence/sqlite/migrations"
)

func TestTransactionManager_CommitsSuccessfulTransaction(t *testing.T) {
	ctx := context.Background()

	db := openTransactionTestDatabase(t)

	manager := NewTransactionManager(db)

	err := manager.WithinTransaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(
			ctx,
			`
			INSERT INTO payments (
                      id,
                      status,
                      version
			) VALUES (?, ?, ?)`,
			"payment-commit",
			"pending",
			1,
		)

		return err
	})
	if err != nil {
		t.Fatalf("WithinTransaction() error = %v", err)
	}

	var count int

	err = db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM payments WHERE id = ?",
		"payment-commit",
	).Scan(&count)
	if err != nil {
		t.Fatalf("count payment: %v", err)
	}

	if count != 1 {
		t.Errorf("payment count = %d, want 1", count)
	}
}

func TestTransactionManager_RollsBackFailedTransaction(t *testing.T) {
	ctx := context.Background()

	db := openTransactionTestDatabase(t)

	manager := NewTransactionManager(db)

	expectedErr := errors.New("operation failed")

	err := manager.WithinTransaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(
			ctx,
			`
				INSERT INTO payments (
                      id,
                      status,
                      version
			  	) VALUES (?, ?, ?)
			`,
			"payment-rollback",
			"pending",
			1,
		)
		if err != nil {
			return err
		}

		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("WithinTransaction() error = %v, want %v", err, expectedErr)
	}

	var count int

	err = db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM payments WHERE id = ?",
		"payment-rollback",
	).Scan(&count)
	if err != nil {
		t.Fatalf("count payment: %v", err)
	}

	if count != 0 {
		t.Errorf("payment count = %d, want 0", count)
	}
}

func TestTransactionManager_RollsBackAllChanges(t *testing.T) {
	ctx := context.Background()

	db := openTransactionTestDatabase(t)

	manager := NewTransactionManager(db)

	expectedErr := errors.New("second operation failed")

	err := manager.WithinTransaction(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(
			ctx,
			`
				INSERT INTO payments (
				                      id,
				                      status,
				                      version
				)
				VALUES (?, ?, ?)
				`,
			"payment-1",
			"pending",
			1,
		)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(
			ctx,
			`
				INSERT INTO payments (
									id,
				                	status,
				                    version
				)
				VALUES (?, ?, ?)
				`,
			"payment-2",
			"pending",
			1,
		)
		if err != nil {
			return err
		}

		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("WithinTransaction() error = %v, want %v", err, expectedErr)
	}

	var count int

	if err := db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM payments WHERE id IN (?, ?)",
		"payment-1",
		"payment-2",
	).Scan(&count); err != nil {
		t.Fatalf("count payment: %v", err)
	}

	if count != 0 {
		t.Errorf("payment count = %d, want 0", count)
	}
}

func openTransactionTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	ctx := context.Background()

	path := filepath.Join(t.TempDir(), "payment-sandbox.db")

	db, err := Open(ctx, config.DatabaseConfig{
		Path:        path,
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := migrations.Up(db); err != nil {
		t.Fatalf("migration.Up() error = %v", err)
	}

	return db
}

package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

type TransactionManager struct {
	db *sql.DB
}

type txContextKey struct{}

func WithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

func TxFromContext(ctx context.Context) *sql.Tx {
	tx, _ := ctx.Value(txContextKey{}).(*sql.Tx)
	return tx
}

func (m *TransactionManager) WithinContext(ctx context.Context, fn func(context.Context) error) error {
	return m.WithinTransaction(ctx, func(tx *sql.Tx) error { return fn(WithTx(ctx, tx)) })
}

func NewTransactionManager(db *sql.DB) *TransactionManager {
	return &TransactionManager{
		db: db,
	}
}

func (m *TransactionManager) WithinTransaction(
	ctx context.Context,
	fn func(tx *sql.Tx) error,
) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf(
				"rollback transaction after error %w: %w",
				err,
				rollbackErr,
			)
		}

		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

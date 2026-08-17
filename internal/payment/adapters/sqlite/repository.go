package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"proxynth/payment-sandbox/internal/payment/application"
	"proxynth/payment-sandbox/internal/payment/domain"
)

var _ application.Repository = (*Repository)(nil)

type executor interface {
	ExecContext(
		ctx context.Context,
		query string,
		args ...any,
	) (sql.Result, error)

	QueryRowContext(
		ctx context.Context,
		query string,
		args ...any,
	) *sql.Row
}

type Repository struct {
	db executor
}

func NewRepository(db executor) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Save(ctx context.Context, payment *domain.Payment) error {
	const query = `
		INSERT INTO payments(
					 id, 
					 status, 
					 version
	   ) VALUES ($1, $2, $3)
	   ON CONFLICT(id) DO UPDATE SET 
		   status = EXCLUDED.status,
		   version = EXCLUDED.version
	   WHERE payments.version = excluded.version - 1`

	result, err := r.db.ExecContext(ctx, query, payment.ID(), payment.Status(), payment.Version())
	if err != nil {
		return fmt.Errorf("save payment %q: %w", payment.ID(), err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"get affected rows for payment %q: %w",
			payment.ID(),
			err,
		)
	}

	if rowsAffected == 0 {
		return fmt.Errorf(
			"save payment %q: %w",
			payment.ID(),
			application.ErrPaymentVersionConflict,
		)
	}

	return nil
}

func (r *Repository) FindByID(ctx context.Context, id domain.ID) (*domain.Payment, error) {
	const query = `
		SELECT
			id,
			status,
			version
		FROM payments
		WHERE id = $1`

	var (
		storedID string
		status   string
		version  uint64
	)

	err := r.db.QueryRowContext(ctx, query, id).Scan(&storedID, &status, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, application.ErrPaymentNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("find payment %q: %w", id, err)
	}

	payment, err := domain.Restore(
		domain.ID(storedID),
		domain.Status(status),
		version,
	)
	if err != nil {
		return nil, fmt.Errorf("restore payment %q: %w", id, err)
	}

	return payment, nil
}

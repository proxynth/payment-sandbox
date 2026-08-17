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

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
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
		   version = EXCLUDED.version`

	_, err := r.db.ExecContext(ctx, query, payment.ID(), payment.Status(), payment.Version())
	if err != nil {
		return fmt.Errorf("save payment %q: %w", payment.ID(), err)
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

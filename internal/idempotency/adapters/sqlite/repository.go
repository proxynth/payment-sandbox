package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"proxynth/payment-sandbox/internal/idempotency/application"
)

type executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Repository struct{ db executor }

func NewRepository(db executor) *Repository { return &Repository{db: db} }

func (r *Repository) Reserve(ctx context.Context, record application.Record) (bool, error) {
	if record.ResponseBody == nil {
		record.ResponseBody = []byte{}
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO idempotency_records(scope,key,fingerprint,status,response_status,response_body)
		VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT(scope,key) DO NOTHING`,
		record.Scope, record.Key, record.Fingerprint, record.Status, record.ResponseStatus, record.ResponseBody)
	if err != nil {
		return false, fmt.Errorf("reserve idempotency record: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("inspect idempotency reservation: %w", err)
	}
	if affected == 1 {
		return true, nil
	}
	existing, err := r.Find(ctx, record.Scope, record.Key)
	if err != nil {
		return false, err
	}
	if existing.Fingerprint != record.Fingerprint {
		return false, application.ErrFingerprintConflict
	}
	return false, nil
}

func (r *Repository) Find(ctx context.Context, scope, key string) (application.Record, error) {
	var record application.Record
	err := r.db.QueryRowContext(ctx, `SELECT scope,key,fingerprint,status,response_status,response_body FROM idempotency_records WHERE scope=$1 AND key=$2`, scope, key).
		Scan(&record.Scope, &record.Key, &record.Fingerprint, &record.Status, &record.ResponseStatus, &record.ResponseBody)
	if errors.Is(err, sql.ErrNoRows) {
		return application.Record{}, application.ErrNotFound
	}
	if err != nil {
		return application.Record{}, fmt.Errorf("find idempotency record: %w", err)
	}
	return record, nil
}

func (r *Repository) Complete(ctx context.Context, record application.Record) error {
	if record.ResponseBody == nil {
		record.ResponseBody = []byte{}
	}
	result, err := r.db.ExecContext(ctx, `UPDATE idempotency_records SET status=$1,response_status=$2,response_body=$3 WHERE scope=$4 AND key=$5 AND fingerprint=$6`, record.Status, record.ResponseStatus, record.ResponseBody, record.Scope, record.Key, record.Fingerprint)
	if err != nil {
		return fmt.Errorf("complete idempotency record: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect idempotency completion: %w", err)
	}
	if affected != 1 {
		return application.ErrNotFound
	}
	return nil
}

func (r *Repository) Release(ctx context.Context, scope, key, fingerprint string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM idempotency_records WHERE scope=$1 AND key=$2 AND fingerprint=$3 AND status=$4`, scope, key, fingerprint, "processing")
	if err != nil {
		return fmt.Errorf("release idempotency record: %w", err)
	}
	return nil
}

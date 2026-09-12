package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	persistencesqlite "proxynth/payment-sandbox/internal/platform/persistence/sqlite"
	"proxynth/payment-sandbox/internal/webhook/application"
)

// DeliveryAuditRepository persists a durable start marker and terminal result
// for each job attempt.
type DeliveryAuditRepository struct{ db executor }

func NewDeliveryAuditRepository(db executor) *DeliveryAuditRepository {
	return &DeliveryAuditRepository{db: db}
}

func (r *DeliveryAuditRepository) Record(ctx context.Context, attempt application.DeliveryAttempt) error {
	if attempt.JobID == "" || attempt.EndpointID == "" || attempt.Outcome == "" {
		return fmt.Errorf("invalid webhook delivery audit attempt")
	}
	if attempt.Outcome == application.DeliveryStarted {
		if tx := persistencesqlite.TxFromContext(ctx); tx != nil {
			return r.recordStart(ctx, tx, attempt)
		}
		if db, ok := r.db.(*sql.DB); ok {
			return persistencesqlite.NewTransactionManager(db).WithinContext(ctx, func(txctx context.Context) error {
				return r.recordStart(txctx, persistencesqlite.TxFromContext(txctx), attempt)
			})
		}
		return r.recordStart(ctx, r.db, attempt)
	}
	return r.recordTerminal(ctx, r.executor(ctx), attempt)
}

func (r *DeliveryAuditRepository) recordStart(ctx context.Context, exec executor, attempt application.DeliveryAttempt) error {
	runtimeSequence, err := persistencesqlite.NextRuntimeSequence(ctx, exec)
	if err != nil {
		return fmt.Errorf("allocate runtime sequence for webhook delivery job %q attempt %d: %w", attempt.JobID, attempt.Attempt, err)
	}
	result, err := exec.ExecContext(ctx, `
		INSERT INTO webhook_delivery_audit(
			job_id, attempt, endpoint_id, correlation_id, causation_id, outcome, http_status, error, runtime_sequence
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(job_id, attempt) DO NOTHING`,
		attempt.JobID, attempt.Attempt, attempt.EndpointID, attempt.CorrelationID,
		attempt.CausationID, attempt.Outcome, attempt.HTTPStatus, attempt.Error, runtimeSequence)
	if err != nil {
		return fmt.Errorf("record webhook delivery audit start for job %q attempt %d: %w", attempt.JobID, attempt.Attempt, err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("record webhook delivery audit start for job %q attempt %d rows affected: %w", attempt.JobID, attempt.Attempt, err)
	}
	if changed == 1 {
		return nil
	}
	var endpointID, correlationID, causationID string
	if err := exec.QueryRowContext(ctx, `
			SELECT endpoint_id, correlation_id, causation_id
			FROM webhook_delivery_audit WHERE job_id = ? AND attempt = ?`, attempt.JobID, attempt.Attempt).
		Scan(&endpointID, &correlationID, &causationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("record webhook delivery audit start for job %q attempt %d: record disappeared", attempt.JobID, attempt.Attempt)
		}
		return fmt.Errorf("inspect webhook delivery audit start for job %q attempt %d: %w", attempt.JobID, attempt.Attempt, err)
	}
	if endpointID != string(attempt.EndpointID) || correlationID != attempt.CorrelationID || causationID != attempt.CausationID {
		return fmt.Errorf("record webhook delivery audit start for job %q attempt %d: immutable metadata already recorded", attempt.JobID, attempt.Attempt)
	}
	return nil
}

func (r *DeliveryAuditRepository) recordTerminal(ctx context.Context, exec executor, attempt application.DeliveryAttempt) error {
	result, err := exec.ExecContext(ctx, `
		UPDATE webhook_delivery_audit
		SET outcome = ?, http_status = ?, error = ?
		WHERE job_id = ? AND attempt = ? AND outcome = ?`,
		attempt.Outcome, attempt.HTTPStatus, attempt.Error, attempt.JobID, attempt.Attempt, application.DeliveryStarted)
	if err != nil {
		return fmt.Errorf("record webhook delivery audit result for job %q attempt %d: %w", attempt.JobID, attempt.Attempt, err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("record webhook delivery audit result for job %q attempt %d rows affected: %w", attempt.JobID, attempt.Attempt, err)
	}
	if changed != 1 {
		var endpointID, correlationID, causationID string
		var outcome string
		var status int
		var recordedError string
		if err := exec.QueryRowContext(ctx, `
			SELECT endpoint_id, correlation_id, causation_id, outcome, http_status, error
			FROM webhook_delivery_audit WHERE job_id = ? AND attempt = ?`, attempt.JobID, attempt.Attempt).
			Scan(&endpointID, &correlationID, &causationID, &outcome, &status, &recordedError); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("record webhook delivery audit result for job %q attempt %d: missing start marker", attempt.JobID, attempt.Attempt)
			}
			return fmt.Errorf("inspect webhook delivery audit result for job %q attempt %d: %w", attempt.JobID, attempt.Attempt, err)
		}
		if endpointID == string(attempt.EndpointID) && correlationID == attempt.CorrelationID && causationID == attempt.CausationID && outcome == string(attempt.Outcome) && status == attempt.HTTPStatus && recordedError == attempt.Error {
			return nil
		}
		return fmt.Errorf("record webhook delivery audit result for job %q attempt %d: terminal result already recorded", attempt.JobID, attempt.Attempt)
	}
	return nil
}

// ListByJob returns the immutable audit trail for one durable delivery job.
func (r *DeliveryAuditRepository) ListByJob(ctx context.Context, jobID string) ([]application.DeliveryAttempt, error) {
	if jobID == "" {
		return nil, fmt.Errorf("webhook delivery audit job ID is required")
	}
	rows, err := r.executor(ctx).QueryContext(ctx, `
		SELECT attempt, endpoint_id, correlation_id, causation_id, outcome, http_status, error, runtime_sequence
		FROM webhook_delivery_audit WHERE job_id = ? ORDER BY CASE WHEN runtime_sequence = 0 THEN 1 ELSE 0 END, runtime_sequence, attempt`, jobID)
	if err != nil {
		return nil, fmt.Errorf("list webhook delivery audit for job %q: %w", jobID, err)
	}
	defer func() { _ = rows.Close() }()

	attempts := make([]application.DeliveryAttempt, 0)
	for rows.Next() {
		var attempt application.DeliveryAttempt
		var number uint64
		if err := rows.Scan(&number, &attempt.EndpointID, &attempt.CorrelationID, &attempt.CausationID, &attempt.Outcome, &attempt.HTTPStatus, &attempt.Error, &attempt.RuntimeSequence); err != nil {
			return nil, fmt.Errorf("scan webhook delivery audit for job %q: %w", jobID, err)
		}
		attempt.JobID = jobID
		attempt.Attempt = number
		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list webhook delivery audit for job %q: %w", jobID, err)
	}
	return attempts, nil
}

func (r *DeliveryAuditRepository) executor(ctx context.Context) executor {
	if tx := persistencesqlite.TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.db
}

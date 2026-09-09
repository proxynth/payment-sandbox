package sqlite

import (
	"context"
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
		_, err := r.executor(ctx).ExecContext(ctx, `
		INSERT INTO webhook_delivery_audit(
			job_id, attempt, endpoint_id, correlation_id, causation_id, outcome, http_status, error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(job_id, attempt) DO NOTHING`,
			attempt.JobID, attempt.Attempt, attempt.EndpointID, attempt.CorrelationID,
			attempt.CausationID, attempt.Outcome, attempt.HTTPStatus, attempt.Error)
		if err != nil {
			return fmt.Errorf("record webhook delivery audit start for job %q attempt %d: %w", attempt.JobID, attempt.Attempt, err)
		}
		return nil
	}
	result, err := r.executor(ctx).ExecContext(ctx, `
		UPDATE webhook_delivery_audit
		SET outcome = ?, http_status = ?, error = ?
		WHERE job_id = ? AND attempt = ?`,
		attempt.Outcome, attempt.HTTPStatus, attempt.Error, attempt.JobID, attempt.Attempt)
	if err != nil {
		return fmt.Errorf("record webhook delivery audit result for job %q attempt %d: %w", attempt.JobID, attempt.Attempt, err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("record webhook delivery audit result for job %q attempt %d rows affected: %w", attempt.JobID, attempt.Attempt, err)
	}
	if changed != 1 {
		return fmt.Errorf("record webhook delivery audit result for job %q attempt %d: missing start marker", attempt.JobID, attempt.Attempt)
	}
	return nil
}

// ListByJob returns the immutable audit trail for one durable delivery job.
func (r *DeliveryAuditRepository) ListByJob(ctx context.Context, jobID string) ([]application.DeliveryAttempt, error) {
	if jobID == "" {
		return nil, fmt.Errorf("webhook delivery audit job ID is required")
	}
	rows, err := r.executor(ctx).QueryContext(ctx, `
		SELECT attempt, endpoint_id, correlation_id, causation_id, outcome, http_status, error
		FROM webhook_delivery_audit WHERE job_id = ? ORDER BY attempt`, jobID)
	if err != nil {
		return nil, fmt.Errorf("list webhook delivery audit for job %q: %w", jobID, err)
	}
	defer rows.Close()

	attempts := make([]application.DeliveryAttempt, 0)
	for rows.Next() {
		var attempt application.DeliveryAttempt
		var number uint64
		if err := rows.Scan(&number, &attempt.EndpointID, &attempt.CorrelationID, &attempt.CausationID, &attempt.Outcome, &attempt.HTTPStatus, &attempt.Error); err != nil {
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

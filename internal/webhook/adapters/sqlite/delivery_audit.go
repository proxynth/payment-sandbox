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

func (r *DeliveryAuditRepository) executor(ctx context.Context) executor {
	if tx := persistencesqlite.TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.db
}

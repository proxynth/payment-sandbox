package sqlite

import (
	"context"
	"fmt"

	persistencesqlite "proxynth/payment-sandbox/internal/platform/persistence/sqlite"
	"proxynth/payment-sandbox/internal/webhook/application"
)

// DeliveryAuditRepository persists one immutable result for each job attempt.
type DeliveryAuditRepository struct{ db executor }

func NewDeliveryAuditRepository(db executor) *DeliveryAuditRepository {
	return &DeliveryAuditRepository{db: db}
}

func (r *DeliveryAuditRepository) Record(ctx context.Context, attempt application.DeliveryAttempt) error {
	if attempt.JobID == "" || attempt.EndpointID == "" || attempt.Outcome == "" {
		return fmt.Errorf("invalid webhook delivery audit attempt")
	}
	_, err := r.executor(ctx).ExecContext(ctx, `
		INSERT INTO webhook_delivery_audit(
			job_id, attempt, endpoint_id, correlation_id, causation_id, outcome, http_status, error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(job_id, attempt) DO NOTHING`,
		attempt.JobID, attempt.Attempt, attempt.EndpointID, attempt.CorrelationID,
		attempt.CausationID, attempt.Outcome, attempt.HTTPStatus, attempt.Error)
	if err != nil {
		return fmt.Errorf("record webhook delivery audit for job %q attempt %d: %w", attempt.JobID, attempt.Attempt, err)
	}
	return nil
}

func (r *DeliveryAuditRepository) executor(ctx context.Context) executor {
	if tx := persistencesqlite.TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.db
}

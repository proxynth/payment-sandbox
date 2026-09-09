package application

import (
	"context"
	"errors"

	webhookapplication "proxynth/payment-sandbox/internal/webhook/application"
)

var ErrNilWebhookDeliveryAudit = errors.New("webhook delivery audit is nil")

// WebhookDeliveryAuditReader is the administration read port for callback
// attempts. It does not expose callback request or response bodies.
type WebhookDeliveryAuditReader interface {
	ListByJob(context.Context, string) ([]webhookapplication.DeliveryAttempt, error)
}

type WebhookAudit struct{ reader WebhookDeliveryAuditReader }

func NewWebhookAudit(reader WebhookDeliveryAuditReader) (*WebhookAudit, error) {
	if reader == nil {
		return nil, ErrNilWebhookDeliveryAudit
	}
	return &WebhookAudit{reader: reader}, nil
}

func (a *WebhookAudit) Execute(ctx context.Context, jobID string) ([]webhookapplication.DeliveryAttempt, error) {
	return a.reader.ListByJob(ctx, jobID)
}

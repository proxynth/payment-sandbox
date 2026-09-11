package application

import (
	"context"

	webhookdomain "proxynth/payment-sandbox/internal/webhook/domain"
)

// DeliveryOutcome is the observed result of one callback attempt.
type DeliveryOutcome string

const (
	// DeliveryStarted means the attempt was durably recorded before the HTTP call.
	// Its terminal result is unknown until it is replaced by a final outcome.
	DeliveryStarted   DeliveryOutcome = "started"
	DeliverySucceeded DeliveryOutcome = "succeeded"
	DeliveryFailed    DeliveryOutcome = "failed"
)

// DeliveryAttempt is an audit-safe description of a callback attempt. It
// deliberately excludes the request and response bodies.
type DeliveryAttempt struct {
	JobID           string
	Attempt         uint64
	EndpointID      webhookdomain.EndpointID
	CorrelationID   string
	CausationID     string
	Outcome         DeliveryOutcome
	HTTPStatus      int
	Error           string
	RuntimeSequence uint64
}

// DeliveryAudit records callback outcomes independently of transport.
type DeliveryAudit interface {
	Record(context.Context, DeliveryAttempt) error
}

// DeliveryAuditReader exposes the durable, body-free delivery history.
type DeliveryAuditReader interface {
	ListByJob(context.Context, string) ([]DeliveryAttempt, error)
}

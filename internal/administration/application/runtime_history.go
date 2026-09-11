package application

import (
	"context"
	"errors"

	paymentapplication "proxynth/payment-sandbox/internal/payment/application"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	schedulerdomain "proxynth/payment-sandbox/internal/scheduler/domain"
	webhookapplication "proxynth/payment-sandbox/internal/webhook/application"
)

var (
	ErrNilRuntimeEvents      = errors.New("runtime history event log is nil")
	ErrNilRuntimeJobs        = errors.New("runtime history job audit is nil")
	ErrNilRuntimeHooks       = errors.New("runtime history webhook audit is nil")
	ErrNilRuntimeTransaction = errors.New("runtime history transaction is nil")
)

type RuntimeJobAuditReader interface {
	ListAuditByAggregate(context.Context, string) ([]schedulerdomain.JobSnapshot, error)
}

type RuntimeHistory struct {
	transaction RuntimeHistoryTransaction
	events      paymentapplication.EventLog
	jobs        RuntimeJobAuditReader
	hooks       webhookapplication.DeliveryAuditReader
}

type PaymentRuntimeHistory struct {
	Payment *paymentdomain.Payment
	Events  []paymentdomain.BusinessEvent
	Jobs    []RuntimeJobHistory
}

type RuntimeJobHistory struct {
	JobID       string
	Type        string
	AggregateID string
	CausationID string
	Snapshots   []schedulerdomain.JobSnapshot
	Deliveries  []webhookapplication.DeliveryAttempt
}

// RuntimeHistoryTransaction gives all readers one consistent database snapshot.
type RuntimeHistoryTransaction interface {
	WithinContext(context.Context, func(context.Context) error) error
}

func NewRuntimeHistory(transaction RuntimeHistoryTransaction, events paymentapplication.EventLog, jobs RuntimeJobAuditReader, hooks webhookapplication.DeliveryAuditReader) (*RuntimeHistory, error) {
	if transaction == nil {
		return nil, ErrNilRuntimeTransaction
	}
	if events == nil {
		return nil, ErrNilRuntimeEvents
	}
	if jobs == nil {
		return nil, ErrNilRuntimeJobs
	}
	if hooks == nil {
		return nil, ErrNilRuntimeHooks
	}
	return &RuntimeHistory{transaction: transaction, events: events, jobs: jobs, hooks: hooks}, nil
}

func (h *RuntimeHistory) Execute(ctx context.Context, id paymentdomain.ID) (PaymentRuntimeHistory, error) {
	var result PaymentRuntimeHistory
	err := h.transaction.WithinContext(ctx, func(txctx context.Context) error {
		var err error
		result, err = h.read(txctx, id)
		return err
	})
	if err != nil {
		return PaymentRuntimeHistory{}, err
	}
	return result, nil
}

func (h *RuntimeHistory) read(ctx context.Context, id paymentdomain.ID) (PaymentRuntimeHistory, error) {
	events, err := h.events.ListByAggregate(ctx, id)
	if err != nil {
		return PaymentRuntimeHistory{}, err
	}
	payment, err := paymentapplication.ReconstructFromEvents(events)
	if err != nil {
		return PaymentRuntimeHistory{}, err
	}
	snapshots, err := h.jobs.ListAuditByAggregate(ctx, string(id))
	if err != nil {
		return PaymentRuntimeHistory{}, err
	}
	groups := make(map[string]*RuntimeJobHistory)
	order := make([]string, 0)
	for _, snapshot := range snapshots {
		jobID := string(snapshot.ID)
		group := groups[jobID]
		if group == nil {
			group = &RuntimeJobHistory{JobID: jobID, Type: string(snapshot.Type), AggregateID: snapshot.AggregateID, CausationID: snapshot.CausationID}
			groups[jobID] = group
			order = append(order, jobID)
		}
		group.Snapshots = append(group.Snapshots, snapshot)
	}
	result := PaymentRuntimeHistory{Payment: payment, Events: events, Jobs: make([]RuntimeJobHistory, 0, len(order))}
	for _, jobID := range order {
		group := groups[jobID]
		group.Deliveries, err = h.hooks.ListByJob(ctx, jobID)
		if err != nil {
			return PaymentRuntimeHistory{}, err
		}
		result.Jobs = append(result.Jobs, *group)
	}
	return result, nil
}

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	paymentapplication "proxynth/payment-sandbox/internal/payment/application"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	"proxynth/payment-sandbox/internal/platform/clock"
	schedulerdomain "proxynth/payment-sandbox/internal/scheduler/domain"
	webhookapplication "proxynth/payment-sandbox/internal/webhook/application"
	webhookdomain "proxynth/payment-sandbox/internal/webhook/domain"
)

type schedulerJobRepository interface {
	Save(context.Context, *schedulerdomain.Job) error
}

type paymentEventPublisher struct {
	events    paymentapplication.EventLog
	endpoints webhookapplication.Repository
	jobs      schedulerJobRepository
	clock     clock.Clock
}

func newPaymentEventPublisher(
	events paymentapplication.EventLog,
	endpoints webhookapplication.Repository,
	jobs schedulerJobRepository,
	businessClock clock.Clock,
) (*paymentEventPublisher, error) {
	if events == nil || endpoints == nil || jobs == nil || businessClock == nil {
		return nil, fmt.Errorf("invalid payment event publisher")
	}
	return &paymentEventPublisher{events: events, endpoints: endpoints, jobs: jobs, clock: businessClock}, nil
}

func (p *paymentEventPublisher) Publish(ctx context.Context, payment *paymentdomain.Payment, eventType paymentdomain.EventType) error {
	if payment == nil {
		return fmt.Errorf("payment event cannot be published for a nil payment")
	}

	at := p.clock.Now().UTC()
	eventID := paymentdomain.EventID(fmt.Sprintf("%s:%s:%d", payment.ID(), eventType, payment.Version()))
	event, err := paymentdomain.NewBusinessEvent(eventID, payment.ID(), eventType, at, payment.Version(), "", "")
	if err != nil {
		return err
	}
	if err := p.events.Append(ctx, event); err != nil {
		return err
	}

	endpoints, err := p.endpoints.List(ctx)
	if err != nil {
		return err
	}
	sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].ID() < endpoints[j].ID() })
	body, err := json.Marshal(paymentWebhookEvent{
		ID: event.ID(), Type: event.Type(), PaymentID: payment.ID(), Amount: payment.Amount().Amount(),
		Currency: payment.Amount().Currency(), Status: payment.Status(), Version: payment.Version(), OccurredAt: at,
	})
	if err != nil {
		return fmt.Errorf("encode payment webhook event: %w", err)
	}

	for _, endpoint := range endpoints {
		if endpoint == nil {
			continue
		}
		if err := p.scheduleDelivery(ctx, endpoint, event, body, at); err != nil {
			return err
		}
	}
	return nil
}

func (p *paymentEventPublisher) scheduleDelivery(
	ctx context.Context,
	endpoint *webhookdomain.Endpoint,
	event paymentdomain.BusinessEvent,
	body []byte,
	at time.Time,
) error {
	payload, err := webhookapplication.NewDeliveryPayload(endpoint.ID(), body)
	if err != nil {
		return err
	}
	jobID := schedulerdomain.JobID(fmt.Sprintf("webhook:%s:%s", event.ID(), endpoint.ID()))
	job, err := schedulerdomain.NewJob(jobID, webhookapplication.DeliveryJobType, payload, at)
	if err != nil {
		return err
	}
	if err := p.jobs.Save(ctx, &job); err != nil {
		return fmt.Errorf("schedule webhook delivery %q: %w", endpoint.ID(), err)
	}
	return nil
}

type paymentWebhookEvent struct {
	ID         paymentdomain.EventID   `json:"id"`
	Type       paymentdomain.EventType `json:"type"`
	PaymentID  paymentdomain.ID        `json:"payment_id"`
	Amount     int64                   `json:"amount"`
	Currency   paymentdomain.Currency  `json:"currency"`
	Status     paymentdomain.Status    `json:"status"`
	Version    uint64                  `json:"version"`
	OccurredAt time.Time               `json:"occurred_at"`
}

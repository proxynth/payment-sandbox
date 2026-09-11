package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	schedulerdomain "proxynth/payment-sandbox/internal/scheduler/domain"
	webhookdomain "proxynth/payment-sandbox/internal/webhook/domain"
)

// DeliveryJobType identifies the durable job handled by OutboundCallback.
const DeliveryJobType = "webhook.delivery"

// DeliveryPayload is the scheduler payload for one outbound callback.
type DeliveryPayload struct {
	EndpointID    webhookdomain.EndpointID `json:"endpoint_id"`
	Body          json.RawMessage          `json:"body"`
	CorrelationID string                   `json:"correlation_id,omitempty"`
	CausationID   string                   `json:"causation_id,omitempty"`
}

// NewDeliveryPayload encodes a callback payload for a webhook delivery job.
func NewDeliveryPayload(endpointID webhookdomain.EndpointID, body []byte, metadata ...string) ([]byte, error) {
	payload := DeliveryPayload{EndpointID: endpointID, Body: append(json.RawMessage(nil), body...)}
	if len(metadata) > 0 {
		payload.CorrelationID = metadata[0]
	}
	if len(metadata) > 1 {
		payload.CausationID = metadata[1]
	}
	if err := validateDeliveryPayload(payload); err != nil {
		return nil, err
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode webhook delivery payload: %w", err)
	}

	return encoded, nil
}

// HTTPClient is the transport boundary used to send callbacks.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// OutboundCallback resolves a webhook endpoint and sends one callback.
type OutboundCallback struct {
	repository Repository
	client     HTTPClient
	audit      DeliveryAudit
}

func NewOutboundCallback(repository Repository, client HTTPClient) (*OutboundCallback, error) {
	if repository == nil {
		return nil, ErrInvalidRepository
	}
	if client == nil {
		return nil, ErrInvalidHTTPClient
	}

	return NewOutboundCallbackWithAudit(repository, client, nil)
}

// NewOutboundCallbackWithAudit creates a callback handler that records each
// completed transport attempt when audit is provided.
func NewOutboundCallbackWithAudit(repository Repository, client HTTPClient, audit DeliveryAudit) (*OutboundCallback, error) {
	if repository == nil {
		return nil, ErrInvalidRepository
	}
	if client == nil {
		return nil, ErrInvalidHTTPClient
	}

	return &OutboundCallback{repository: repository, client: client, audit: audit}, nil
}

// Execute handles one DeliveryJobType payload. It does not retry or persist
// the job; those responsibilities belong to the Runtime context.
func (d *OutboundCallback) Execute(ctx context.Context, payload []byte) error {
	delivery, err := decodeDeliveryPayload(payload)
	if err != nil {
		return err
	}
	if err := d.start(ctx, delivery); err != nil {
		return err
	}

	endpoint, err := d.repository.FindByID(ctx, delivery.EndpointID)
	if err != nil {
		return d.finish(ctx, delivery, DeliveryFailed, 0, err)
	}
	if endpoint == nil {
		return d.finish(ctx, delivery, DeliveryFailed, 0, ErrEndpointNotFound)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL(), bytes.NewReader(delivery.Body))
	if err != nil {
		return d.finish(ctx, delivery, DeliveryFailed, 0, fmt.Errorf("%w: create request: %w", ErrCallbackDeliveryFailed, err))
	}
	request.Header.Set("Content-Type", "application/json")
	if delivery.CorrelationID != "" {
		request.Header.Set("X-Correlation-ID", delivery.CorrelationID)
	}
	if delivery.CausationID != "" {
		request.Header.Set("X-Causation-ID", delivery.CausationID)
	}

	response, err := d.client.Do(request)
	if err != nil {
		return d.finish(ctx, delivery, DeliveryFailed, 0, fmt.Errorf("%w: %w", ErrCallbackDeliveryFailed, err))
	}
	if response == nil {
		return d.finish(ctx, delivery, DeliveryFailed, 0, fmt.Errorf("%w: HTTP client returned nil response", ErrCallbackDeliveryFailed))
	}
	if response.Body == nil {
		return d.finish(ctx, delivery, DeliveryFailed, response.StatusCode, fmt.Errorf("%w: HTTP client returned nil response body", ErrCallbackDeliveryFailed))
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return d.finish(ctx, delivery, DeliveryFailed, response.StatusCode, fmt.Errorf("%w: unexpected HTTP status %d", ErrCallbackDeliveryFailed, response.StatusCode))
	}

	return d.finish(ctx, delivery, DeliverySucceeded, response.StatusCode, nil)
}

func (d *OutboundCallback) finish(ctx context.Context, delivery DeliveryPayload, outcome DeliveryOutcome, status int, executionErr error) error {
	if d.audit == nil {
		return executionErr
	}

	attemptRecord := d.newAttempt(ctx, delivery, outcome, status)
	if executionErr != nil {
		attemptRecord.Error = executionErr.Error()
	}
	if err := d.audit.Record(ctx, attemptRecord); err != nil {
		if executionErr != nil {
			return fmt.Errorf("%w; record webhook delivery audit: %w", executionErr, err)
		}
		return fmt.Errorf("record webhook delivery audit: %w", err)
	}
	return executionErr
}

func (d *OutboundCallback) start(ctx context.Context, delivery DeliveryPayload) error {
	if d.audit == nil {
		return nil
	}
	if err := d.audit.Record(ctx, d.newAttempt(ctx, delivery, DeliveryStarted, 0)); err != nil {
		return fmt.Errorf("record webhook delivery attempt: %w", err)
	}
	return nil
}

func (d *OutboundCallback) newAttempt(ctx context.Context, delivery DeliveryPayload, outcome DeliveryOutcome, status int) DeliveryAttempt {
	metadata, ok := schedulerdomain.ExecutionMetadataFromContext(ctx)
	jobID := "direct:" + delivery.CausationID
	attempt := uint64(0)
	if ok {
		jobID = string(metadata.JobID)
		attempt = metadata.Attempt
	}
	return DeliveryAttempt{
		JobID:         jobID,
		Attempt:       attempt,
		EndpointID:    delivery.EndpointID,
		CorrelationID: delivery.CorrelationID,
		CausationID:   delivery.CausationID,
		Outcome:       outcome,
		HTTPStatus:    status,
	}
}

func decodeDeliveryPayload(payload []byte) (DeliveryPayload, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	var delivery DeliveryPayload
	if err := decoder.Decode(&delivery); err != nil {
		return DeliveryPayload{}, ErrInvalidDeliveryPayload
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return DeliveryPayload{}, ErrInvalidDeliveryPayload
	}
	if err := validateDeliveryPayload(delivery); err != nil {
		return DeliveryPayload{}, err
	}

	delivery.Body = append(json.RawMessage(nil), delivery.Body...)
	return delivery, nil
}

func validateDeliveryPayload(payload DeliveryPayload) error {
	if payload.EndpointID == "" || len(payload.Body) == 0 || !json.Valid(payload.Body) {
		return ErrInvalidDeliveryPayload
	}

	return nil
}

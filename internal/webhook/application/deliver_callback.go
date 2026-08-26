package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
}

func NewOutboundCallback(repository Repository, client HTTPClient) (*OutboundCallback, error) {
	if repository == nil {
		return nil, ErrInvalidRepository
	}
	if client == nil {
		return nil, ErrInvalidHTTPClient
	}

	return &OutboundCallback{repository: repository, client: client}, nil
}

// Execute handles one DeliveryJobType payload. It does not retry or persist
// the job; those responsibilities belong to the Runtime context.
func (d *OutboundCallback) Execute(ctx context.Context, payload []byte) error {
	delivery, err := decodeDeliveryPayload(payload)
	if err != nil {
		return err
	}

	endpoint, err := d.repository.FindByID(ctx, delivery.EndpointID)
	if err != nil {
		return err
	}
	if endpoint == nil {
		return ErrEndpointNotFound
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL(), bytes.NewReader(delivery.Body))
	if err != nil {
		return fmt.Errorf("%w: create request: %w", ErrCallbackDeliveryFailed, err)
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
		return fmt.Errorf("%w: %w", ErrCallbackDeliveryFailed, err)
	}
	if response == nil {
		return fmt.Errorf("%w: HTTP client returned nil response", ErrCallbackDeliveryFailed)
	}
	if response.Body == nil {
		return fmt.Errorf("%w: HTTP client returned nil response body", ErrCallbackDeliveryFailed)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: unexpected HTTP status %d", ErrCallbackDeliveryFailed, response.StatusCode)
	}

	return nil
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

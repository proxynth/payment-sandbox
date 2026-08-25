package http

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	administrationapplication "proxynth/payment-sandbox/internal/administration/application"
	"proxynth/payment-sandbox/internal/api"
	paymentapplication "proxynth/payment-sandbox/internal/payment/application"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
)

const paymentTimelinePathPrefix = "/admin/payments/"

type TimelineHandler struct {
	timeline *administrationapplication.Timeline
}

func NewTimelineHandler(
	payments paymentapplication.Repository,
	events paymentapplication.EventLog,
) (*TimelineHandler, error) {
	timeline, err := administrationapplication.NewTimeline(payments, events)
	if err != nil {
		return nil, err
	}

	return &TimelineHandler{timeline: timeline}, nil
}

func (h *TimelineHandler) Register(server *api.Server, token string) error {
	return server.HandleAdminPrefix(http.MethodGet, paymentTimelinePathPrefix, http.HandlerFunc(h.getTimeline), token)
}

type timelineResponse struct {
	Payment paymentResponse `json:"payment"`
	Events  []eventResponse `json:"events"`
}

type paymentResponse struct {
	ID               string `json:"id"`
	Amount           int64  `json:"amount"`
	Currency         string `json:"currency"`
	Status           string `json:"status"`
	AuthorizedAmount int64  `json:"authorized_amount"`
	CapturedAmount   int64  `json:"captured_amount"`
	RefundedAmount   int64  `json:"refunded_amount"`
	Version          uint64 `json:"version"`
}

type eventResponse struct {
	ID               string `json:"id"`
	AggregateID      string `json:"aggregate_id"`
	Type             string `json:"type"`
	OccurredAt       string `json:"occurred_at"`
	AggregateVersion uint64 `json:"aggregate_version"`
	CorrelationID    string `json:"correlation_id"`
	CausationID      string `json:"causation_id"`
}

func (h *TimelineHandler) getTimeline(writer http.ResponseWriter, request *http.Request) {
	id, ok := timelinePaymentID(request.URL.Path)
	if !ok {
		api.WriteError(writer, http.StatusNotFound, "not_found", "payment timeline route not found")
		return
	}

	timeline, err := h.timeline.Execute(request.Context(), paymentdomain.ID(id))
	if err != nil {
		writeTimelineError(writer, err)
		return
	}

	events := make([]eventResponse, 0, len(timeline.Events))
	for _, event := range timeline.Events {
		events = append(events, eventResponse{
			ID:               string(event.ID()),
			AggregateID:      string(event.AggregateID()),
			Type:             string(event.Type()),
			OccurredAt:       event.OccurredAt().UTC().Format(time.RFC3339Nano),
			AggregateVersion: event.AggregateVersion(),
			CorrelationID:    event.CorrelationID(),
			CausationID:      string(event.CausationID()),
		})
	}

	api.WriteJSON(writer, http.StatusOK, timelineResponse{
		Payment: newTimelinePaymentResponse(timeline.Payment),
		Events:  events,
	})
}

func timelinePaymentID(path string) (string, bool) {
	parts := strings.Split(strings.TrimPrefix(path, paymentTimelinePathPrefix), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "timeline" {
		return "", false
	}

	id, err := url.PathUnescape(parts[0])
	return id, err == nil && id != ""
}

func newTimelinePaymentResponse(payment *paymentdomain.Payment) paymentResponse {
	return paymentResponse{
		ID:               string(payment.ID()),
		Amount:           payment.Amount().Amount(),
		Currency:         string(payment.Amount().Currency()),
		Status:           string(payment.Status()),
		AuthorizedAmount: payment.AuthorizedAmount().Amount(),
		CapturedAmount:   payment.CapturedAmount().Amount(),
		RefundedAmount:   payment.RefundedAmount().Amount(),
		Version:          payment.Version(),
	}
}

func writeTimelineError(writer http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	if errors.Is(err, paymentapplication.ErrPaymentNotFound) {
		status = http.StatusNotFound
		code = "payment_not_found"
	}

	api.WriteError(writer, status, code, err.Error())
}

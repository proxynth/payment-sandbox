package http

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"proxynth/payment-sandbox/internal/administration/application"
	"proxynth/payment-sandbox/internal/api"
	paymentapplication "proxynth/payment-sandbox/internal/payment/application"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
)

const diagnosticsPaymentPathPrefix = "/admin/diagnostics/payments/"

type DiagnosticsHandler struct {
	diagnostics *application.Diagnostics
}

func NewDiagnosticsHandler(
	payments paymentapplication.Repository,
	events paymentapplication.EventLog,
	clockSource interface{ Now() time.Time },
	providers interface {
		IDs() []providerdomain.ProviderID
	},
) (*DiagnosticsHandler, error) {
	diagnostics, err := application.NewDiagnostics(payments, events, clockSource, providers)
	if err != nil {
		return nil, err
	}

	return &DiagnosticsHandler{diagnostics: diagnostics}, nil
}

func (h *DiagnosticsHandler) Register(server *api.Server, token string) error {
	return server.HandleAdminPrefix(http.MethodGet, diagnosticsPaymentPathPrefix, http.HandlerFunc(h.getDiagnostics), token)
}

type diagnosticsResponse struct {
	Payment     paymentResponse `json:"payment"`
	Events      []eventResponse `json:"events"`
	CurrentTime string          `json:"current_time"`
	ProviderIDs []string        `json:"providers"`
}

func (h *DiagnosticsHandler) getDiagnostics(writer http.ResponseWriter, request *http.Request) {
	id, ok := diagnosticsPaymentID(request.URL.Path)
	if !ok {
		api.WriteError(writer, http.StatusNotFound, "not_found", "diagnostics route not found")
		return
	}

	diagnostics, err := h.diagnostics.Execute(request.Context(), paymentdomain.ID(id))
	if err != nil {
		writeDiagnosticsError(writer, err)
		return
	}

	events := make([]eventResponse, 0, len(diagnostics.Timeline.Events))
	for _, event := range diagnostics.Timeline.Events {
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

	providers := make([]string, 0, len(diagnostics.ProviderIDs))
	for _, providerID := range diagnostics.ProviderIDs {
		providers = append(providers, string(providerID))
	}

	api.WriteJSON(writer, http.StatusOK, diagnosticsResponse{
		Payment:     newTimelinePaymentResponse(diagnostics.Timeline.Payment),
		Events:      events,
		CurrentTime: diagnostics.CurrentTime.Format(time.RFC3339Nano),
		ProviderIDs: providers,
	})
}

func diagnosticsPaymentID(path string) (string, bool) {
	parts := strings.Split(strings.TrimPrefix(path, diagnosticsPaymentPathPrefix), "/")
	if len(parts) != 1 || parts[0] == "" {
		return "", false
	}

	id, err := url.PathUnescape(parts[0])
	return id, err == nil && id != ""
}

func writeDiagnosticsError(writer http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	if errors.Is(err, paymentapplication.ErrPaymentNotFound) {
		status = http.StatusNotFound
		code = "payment_not_found"
	}

	api.WriteError(writer, status, code, err.Error())
}

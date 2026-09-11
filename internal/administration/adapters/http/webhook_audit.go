package http

import (
	"net/http"
	"net/url"
	"strings"

	administrationapplication "proxynth/payment-sandbox/internal/administration/application"
	"proxynth/payment-sandbox/internal/api"
	webhookapplication "proxynth/payment-sandbox/internal/webhook/application"
)

const webhookAuditPathPrefix = "/admin/webhook-jobs/"

type WebhookAuditHandler struct {
	audit *administrationapplication.WebhookAudit
}

func NewWebhookAuditHandler(reader administrationapplication.WebhookDeliveryAuditReader) (*WebhookAuditHandler, error) {
	audit, err := administrationapplication.NewWebhookAudit(reader)
	if err != nil {
		return nil, err
	}
	return &WebhookAuditHandler{audit: audit}, nil
}

func (h *WebhookAuditHandler) Register(server *api.Server, token string) error {
	return server.HandleAdminPrefix(http.MethodGet, webhookAuditPathPrefix, http.HandlerFunc(h.getAudit), token)
}

type webhookAuditResponse struct {
	JobID    string                   `json:"job_id"`
	Attempts []webhookAttemptResponse `json:"attempts"`
}

type webhookAttemptResponse struct {
	Attempt       uint64 `json:"attempt"`
	EndpointID    string `json:"endpoint_id"`
	CorrelationID string `json:"correlation_id"`
	CausationID   string `json:"causation_id"`
	Outcome       string `json:"outcome"`
	HTTPStatus    int    `json:"http_status"`
	Error         string `json:"error,omitempty"`
}

func (h *WebhookAuditHandler) getAudit(writer http.ResponseWriter, request *http.Request) {
	jobID, ok := webhookAuditJobID(request.URL.Path)
	if !ok {
		api.WriteError(writer, http.StatusNotFound, "not_found", "webhook audit route not found")
		return
	}
	attempts, err := h.audit.Execute(request.Context(), jobID)
	if err != nil {
		api.WriteError(writer, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	response := webhookAuditResponse{JobID: jobID, Attempts: make([]webhookAttemptResponse, 0, len(attempts))}
	for _, attempt := range attempts {
		response.Attempts = append(response.Attempts, newWebhookAttemptResponse(attempt))
	}
	api.WriteJSON(writer, http.StatusOK, response)
}

func webhookAuditJobID(path string) (string, bool) {
	parts := strings.Split(strings.TrimPrefix(path, webhookAuditPathPrefix), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "deliveries" {
		return "", false
	}
	jobID, err := url.PathUnescape(parts[0])
	return jobID, err == nil && jobID != ""
}

func newWebhookAttemptResponse(attempt webhookapplication.DeliveryAttempt) webhookAttemptResponse {
	return webhookAttemptResponse{
		Attempt: attempt.Attempt, EndpointID: string(attempt.EndpointID), CorrelationID: attempt.CorrelationID,
		CausationID: attempt.CausationID, Outcome: string(attempt.Outcome), HTTPStatus: attempt.HTTPStatus, Error: attempt.Error,
	}
}

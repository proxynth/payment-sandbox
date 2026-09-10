package http

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	administrationapplication "proxynth/payment-sandbox/internal/administration/application"
	"proxynth/payment-sandbox/internal/api"
	paymentapplication "proxynth/payment-sandbox/internal/payment/application"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	webhookapplication "proxynth/payment-sandbox/internal/webhook/application"
)

const runtimeHistoryPathPrefix = "/admin/runtime-history/payments/"

type RuntimeHistoryHandler struct {
	history *administrationapplication.RuntimeHistory
}

func NewRuntimeHistoryHandler(transaction administrationapplication.RuntimeHistoryTransaction, events paymentapplication.EventLog, jobs administrationapplication.RuntimeJobAuditReader, hooks webhookapplication.DeliveryAuditReader) (*RuntimeHistoryHandler, error) {
	history, err := administrationapplication.NewRuntimeHistory(transaction, events, jobs, hooks)
	if err != nil {
		return nil, err
	}
	return &RuntimeHistoryHandler{history: history}, nil
}

type runtimeHistoryResponse struct {
	Payment paymentResponse      `json:"payment"`
	Events  []eventResponse      `json:"events"`
	Jobs    []runtimeJobResponse `json:"jobs"`
}
type runtimeJobResponse struct {
	JobID       string                    `json:"job_id"`
	Type        string                    `json:"type"`
	AggregateID string                    `json:"aggregate_id,omitempty"`
	CausationID string                    `json:"causation_id,omitempty"`
	Snapshots   []runtimeSnapshotResponse `json:"snapshots"`
	Deliveries  []webhookAttemptResponse  `json:"deliveries"`
}
type runtimeSnapshotResponse struct {
	Status        string `json:"status"`
	Attempts      uint64 `json:"attempts"`
	ScheduledAt   string `json:"scheduled_at"`
	NextAttemptAt string `json:"next_attempt_at"`
	LeaseOwner    string `json:"lease_owner,omitempty"`
}

func (h *RuntimeHistoryHandler) Register(server *api.Server, token string) error {
	return server.HandleAdminPrefix(http.MethodGet, runtimeHistoryPathPrefix, http.HandlerFunc(h.getHistory), token)
}

func (h *RuntimeHistoryHandler) getHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := runtimeHistoryPaymentID(r.URL.Path)
	if !ok {
		api.WriteError(w, http.StatusNotFound, "not_found", "runtime history route not found")
		return
	}
	history, err := h.history.Execute(r.Context(), paymentdomain.ID(id))
	if err != nil {
		writeTimelineError(w, err)
		return
	}
	events := make([]eventResponse, 0, len(history.Events))
	for _, e := range history.Events {
		events = append(events, eventResponse{ID: string(e.ID()), AggregateID: string(e.AggregateID()), Type: string(e.Type()), OccurredAt: e.OccurredAt().UTC().Format(time.RFC3339Nano), AggregateVersion: e.AggregateVersion(), CorrelationID: e.CorrelationID(), CausationID: string(e.CausationID())})
	}
	jobs := make([]runtimeJobResponse, 0, len(history.Jobs))
	for _, j := range history.Jobs {
		out := runtimeJobResponse{JobID: j.JobID, Type: j.Type, AggregateID: j.AggregateID, CausationID: j.CausationID, Snapshots: make([]runtimeSnapshotResponse, 0, len(j.Snapshots)), Deliveries: make([]webhookAttemptResponse, 0, len(j.Deliveries))}
		for _, s := range j.Snapshots {
			out.Snapshots = append(out.Snapshots, runtimeSnapshotResponse{Status: string(s.Status), Attempts: s.Attempts, ScheduledAt: s.ScheduledAt.UTC().Format(time.RFC3339Nano), NextAttemptAt: s.NextAttemptAt.UTC().Format(time.RFC3339Nano), LeaseOwner: s.LeaseOwner})
		}
		for _, d := range j.Deliveries {
			out.Deliveries = append(out.Deliveries, newWebhookAttemptResponse(d))
		}
		jobs = append(jobs, out)
	}
	api.WriteJSON(w, http.StatusOK, runtimeHistoryResponse{Payment: newTimelinePaymentResponse(history.Payment), Events: events, Jobs: jobs})
}

func runtimeHistoryPaymentID(path string) (string, bool) {
	parts := strings.Split(strings.TrimPrefix(path, runtimeHistoryPathPrefix), "/")
	if len(parts) != 1 || parts[0] == "" {
		return "", false
	}
	id, err := url.PathUnescape(parts[0])
	return id, err == nil && id != ""
}

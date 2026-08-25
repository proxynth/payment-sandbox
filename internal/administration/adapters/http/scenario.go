package http

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"proxynth/payment-sandbox/internal/administration/application"
	"proxynth/payment-sandbox/internal/api"
	replaydomain "proxynth/payment-sandbox/internal/replay/domain"
)

const scenarioPathPrefix = "/admin/scenarios/"

type ScenarioHandler struct {
	inspection *application.ScenarioInspection
}

func NewScenarioHandler(repository application.ScenarioRepository) (*ScenarioHandler, error) {
	inspection, err := application.NewScenarioInspection(repository)
	if err != nil {
		return nil, err
	}

	return &ScenarioHandler{inspection: inspection}, nil
}

func (h *ScenarioHandler) Register(server *api.Server, token string) error {
	return server.HandleAdminPrefix(http.MethodGet, scenarioPathPrefix, http.HandlerFunc(h.getScenario), token)
}

type scenarioResponse struct {
	ID                  string                     `json:"id"`
	Provider            providerConfiguration      `json:"provider"`
	InitialVirtualTime  time.Time                  `json:"initial_virtual_time"`
	DeterministicConfig deterministicConfiguration `json:"deterministic_configuration"`
	InitialPayments     []initialPayment           `json:"initial_payments"`
	Commands            []scenarioCommand          `json:"commands"`
}

type providerConfiguration struct {
	ID string `json:"id"`
}

type deterministicConfiguration struct {
	Seed uint64 `json:"seed"`
}

type initialPayment struct {
	ID               string `json:"id"`
	Amount           int64  `json:"amount"`
	Currency         string `json:"currency"`
	Status           string `json:"status"`
	AuthorizedAmount int64  `json:"authorized_amount"`
	CapturedAmount   int64  `json:"captured_amount"`
	RefundedAmount   int64  `json:"refunded_amount"`
	Version          uint64 `json:"version"`
}

type scenarioCommand struct {
	Type      string `json:"type"`
	PaymentID string `json:"payment_id"`
	Amount    int64  `json:"amount,omitempty"`
	Currency  string `json:"currency,omitempty"`
}

func (h *ScenarioHandler) getScenario(writer http.ResponseWriter, request *http.Request) {
	id, ok := scenarioID(request.URL.Path)
	if !ok {
		api.WriteError(writer, http.StatusNotFound, "not_found", "scenario route not found")
		return
	}

	scenario, err := h.inspection.Execute(request.Context(), replaydomain.ScenarioID(id))
	if err != nil {
		writeScenarioError(writer, err)
		return
	}

	payments := make([]initialPayment, 0, len(scenario.InitialPayments))
	for _, payment := range scenario.InitialPayments {
		payments = append(payments, initialPayment{
			ID:               string(payment.ID),
			Amount:           payment.Amount.Amount(),
			Currency:         string(payment.Amount.Currency()),
			Status:           string(payment.Status),
			AuthorizedAmount: payment.AuthorizedAmount,
			CapturedAmount:   payment.CapturedAmount,
			RefundedAmount:   payment.RefundedAmount,
			Version:          payment.Version,
		})
	}

	commands := make([]scenarioCommand, 0, len(scenario.Commands))
	for _, command := range scenario.Commands {
		commands = append(commands, scenarioCommand{
			Type:      string(command.Type),
			PaymentID: string(command.PaymentID),
			Amount:    command.Amount.Amount(),
			Currency:  string(command.Amount.Currency()),
		})
	}

	api.WriteJSON(writer, http.StatusOK, scenarioResponse{
		ID:                 string(scenario.ID),
		Provider:           providerConfiguration{ID: string(scenario.Provider.ID)},
		InitialVirtualTime: scenario.InitialVirtualTime.UTC(),
		DeterministicConfig: deterministicConfiguration{
			Seed: scenario.DeterministicConfiguration.Seed,
		},
		InitialPayments: payments,
		Commands:        commands,
	})
}

func scenarioID(path string) (string, bool) {
	parts := strings.Split(strings.TrimPrefix(path, scenarioPathPrefix), "/")
	if len(parts) != 1 || parts[0] == "" {
		return "", false
	}

	id, err := url.PathUnescape(parts[0])
	return id, err == nil && id != ""
}

func writeScenarioError(writer http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	switch {
	case errors.Is(err, application.ErrScenarioNotFound):
		status = http.StatusNotFound
		code = "scenario_not_found"
	case errors.Is(err, replaydomain.ErrInvalidScenarioID),
		errors.Is(err, replaydomain.ErrInvalidScenarioTime),
		errors.Is(err, replaydomain.ErrInvalidProviderConfiguration),
		errors.Is(err, replaydomain.ErrDuplicateInitialPayment),
		errors.Is(err, replaydomain.ErrInvalidCommand):
		status = http.StatusBadRequest
		code = "invalid_request"
	}

	api.WriteError(writer, status, code, err.Error())
}

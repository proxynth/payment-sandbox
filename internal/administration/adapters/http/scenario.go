package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"proxynth/payment-sandbox/internal/administration/application"
	"proxynth/payment-sandbox/internal/api"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
	replayapplication "proxynth/payment-sandbox/internal/replay/application"
	replaydomain "proxynth/payment-sandbox/internal/replay/domain"
)

const scenarioPathPrefix = "/admin/scenarios/"

type ScenarioHandler struct {
	inspection *application.ScenarioInspection
	service    *replayapplication.ScenarioService
}

func NewScenarioHandler(repository application.ScenarioRepository, service *replayapplication.ScenarioService) (*ScenarioHandler, error) {
	inspection, err := application.NewScenarioInspection(repository)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, errors.New("scenario service is nil")
	}

	return &ScenarioHandler{inspection: inspection, service: service}, nil
}

func (h *ScenarioHandler) Register(server *api.Server, token string) error {
	if err := server.HandleAdmin(http.MethodPost, "/admin/scenarios", http.HandlerFunc(h.createScenario), token); err != nil {
		return err
	}
	if err := server.HandleAdminPrefix(http.MethodGet, scenarioPathPrefix, http.HandlerFunc(h.getScenario), token); err != nil {
		return err
	}
	return server.HandleAdminPrefix(http.MethodPost, scenarioPathPrefix, http.HandlerFunc(h.executeScenario), token)
}

type scenarioRequest struct {
	ID       string `json:"id"`
	Provider struct {
		ID      string `json:"id"`
		Profile string `json:"profile"`
	} `json:"provider"`
	InitialVirtualTime         time.Time `json:"initial_virtual_time"`
	DeterministicConfiguration struct {
		Seed uint64 `json:"seed"`
	} `json:"deterministic_configuration"`
	InitialPayments []paymentRequestState `json:"initial_payments"`
	Commands        []commandRequest      `json:"commands"`
}

type paymentRequestState struct {
	ID               string `json:"id"`
	Amount           int64  `json:"amount"`
	Currency         string `json:"currency"`
	Status           string `json:"status"`
	AuthorizedAmount int64  `json:"authorized_amount"`
	CapturedAmount   int64  `json:"captured_amount"`
	RefundedAmount   int64  `json:"refunded_amount"`
	Version          uint64 `json:"version"`
}

type commandRequest struct {
	Type        string `json:"type"`
	PaymentID   string `json:"payment_id"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Duration    string `json:"duration"`
	OperationID string `json:"operation_id"`
}

func (h *ScenarioHandler) createScenario(writer http.ResponseWriter, request *http.Request) {
	if h.service == nil {
		api.WriteError(writer, http.StatusNotImplemented, "not_implemented", "scenario creation is not configured")
		return
	}
	var input scenarioRequest
	if !decodeScenarioJSON(writer, request, &input) {
		return
	}
	scenario, err := input.scenario()
	if err != nil {
		writeScenarioError(writer, err)
		return
	}
	if err := h.service.Create(request.Context(), scenario); err != nil {
		writeScenarioError(writer, err)
		return
	}
	h.writeScenario(writer, http.StatusCreated, scenario)
}

func (h *ScenarioHandler) executeScenario(writer http.ResponseWriter, request *http.Request) {
	if h.service == nil {
		api.WriteError(writer, http.StatusNotImplemented, "not_implemented", "scenario execution is not configured")
		return
	}
	id, ok := scenarioExecutionID(request.URL.Path)
	if !ok {
		api.WriteError(writer, http.StatusNotFound, "not_found", "scenario route not found")
		return
	}
	result, err := h.service.Execute(request.Context(), replaydomain.ScenarioID(id))
	if err != nil {
		writeScenarioError(writer, err)
		return
	}
	api.WriteJSON(writer, http.StatusOK, newScenarioExecutionResponse(result))
}

type scenarioExecutionResponse struct {
	ScenarioID          string                     `json:"scenario_id"`
	Provider            providerConfiguration      `json:"provider"`
	DeterministicConfig deterministicConfiguration `json:"deterministic_configuration"`
	CurrentVirtualTime  time.Time                  `json:"current_virtual_time"`
	Payments            []initialPayment           `json:"payments"`
	AsyncOperations     any                        `json:"async_operations"`
}

func newScenarioExecutionResponse(result replayapplication.Result) scenarioExecutionResponse {
	payments := make([]initialPayment, 0, len(result.Payments))
	for _, payment := range result.Payments {
		payments = append(payments, initialPayment{ID: string(payment.ID), Amount: payment.Amount.Amount(), Currency: string(payment.Amount.Currency()), Status: string(payment.Status), AuthorizedAmount: payment.AuthorizedAmount, CapturedAmount: payment.CapturedAmount, RefundedAmount: payment.RefundedAmount, Version: payment.Version})
	}
	return scenarioExecutionResponse{ScenarioID: string(result.ScenarioID), Provider: providerConfiguration{ID: string(result.Provider.ID), Profile: result.Provider.Profile}, DeterministicConfig: deterministicConfiguration{Seed: result.DeterministicConfiguration.Seed}, CurrentVirtualTime: result.CurrentVirtualTime.UTC(), Payments: payments, AsyncOperations: result.AsyncOperations}
}

func decodeScenarioJSON(writer http.ResponseWriter, request *http.Request, destination any) bool {
	if !strings.HasPrefix(request.Header.Get("Content-Type"), "application/json") || request.Body == nil {
		api.WriteError(writer, http.StatusBadRequest, "invalid_request", "a JSON request body is required")
		return false
	}
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		api.WriteError(writer, http.StatusBadRequest, "invalid_request", "invalid JSON request body")
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		api.WriteError(writer, http.StatusBadRequest, "invalid_request", "request body must contain one JSON object")
		return false
	}
	return true
}

func (input scenarioRequest) scenario() (*replaydomain.Scenario, error) {
	payments := make([]paymentdomain.PaymentState, 0, len(input.InitialPayments))
	for _, item := range input.InitialPayments {
		payments = append(payments, paymentdomain.PaymentState{ID: paymentdomain.ID(item.ID), Amount: mustMoney(item.Amount, item.Currency), Status: paymentdomain.Status(item.Status), AuthorizedAmount: item.AuthorizedAmount, CapturedAmount: item.CapturedAmount, RefundedAmount: item.RefundedAmount, Version: item.Version})
	}
	commands := make([]replaydomain.Command, 0, len(input.Commands))
	for _, item := range input.Commands {
		amount := mustMoney(item.Amount, item.Currency)
		var duration time.Duration
		var err error
		if item.Duration != "" {
			duration, err = time.ParseDuration(item.Duration)
			if err != nil {
				return nil, replaydomain.ErrInvalidCommand
			}
		}
		commands = append(commands, replaydomain.Command{Type: replaydomain.CommandType(item.Type), PaymentID: paymentdomain.ID(item.PaymentID), Amount: amount, Duration: duration, OperationID: item.OperationID})
	}
	return replaydomain.New(replaydomain.ScenarioID(input.ID), payments, commands, replaydomain.ProviderConfiguration{ID: providerdomain.ProviderID(input.Provider.ID), Profile: input.Provider.Profile}, input.InitialVirtualTime, replaydomain.DeterministicConfiguration{Seed: input.DeterministicConfiguration.Seed})
}

func mustMoney(amount int64, currency string) paymentdomain.Money {
	money, _ := paymentdomain.NewMoney(amount, paymentdomain.Currency(currency))
	return money
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
	ID      string `json:"id"`
	Profile string `json:"profile,omitempty"`
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

	h.writeScenario(writer, http.StatusOK, scenario)
}

func (h *ScenarioHandler) writeScenario(writer http.ResponseWriter, status int, scenario *replaydomain.Scenario) {
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

	api.WriteJSON(writer, status, scenarioResponse{
		ID:                 string(scenario.ID),
		Provider:           providerConfiguration{ID: string(scenario.Provider.ID), Profile: scenario.Provider.Profile},
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

func scenarioExecutionID(path string) (string, bool) {
	parts := strings.Split(strings.TrimPrefix(path, scenarioPathPrefix), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "execute" {
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
	case errors.Is(err, replayapplication.ErrScenarioNotFound):
		status = http.StatusNotFound
		code = "scenario_not_found"
	case errors.Is(err, replaydomain.ErrScenarioAlreadyExists):
		status = http.StatusConflict
		code = "scenario_already_exists"
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

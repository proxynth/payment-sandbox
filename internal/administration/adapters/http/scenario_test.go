package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/api"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	replayapplication "proxynth/payment-sandbox/internal/replay/application"
	replaydomain "proxynth/payment-sandbox/internal/replay/domain"
)

func TestScenarioHandlerReturnsStructuredScenario(t *testing.T) {
	scenario := httpScenario(t)
	handler, err := NewScenarioHandler(&httpScenarioRepository{scenario: scenario})
	if err != nil {
		t.Fatalf("NewScenarioHandler() error = %v", err)
	}
	server := scenarioServer(t, handler)

	response := httptest.NewRecorder()
	server.ServeHTTP(response, adminRequest(http.MethodGet, "/admin/scenarios/scenario-http"))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	for _, expected := range []string{
		`"id":"scenario-http"`, `"id":"stripe"`, `"seed":99`,
		`"id":"payment-initial"`, `"type":"create_payment"`, `"payment_id":"payment-created"`,
	} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("body = %q, missing %q", response.Body.String(), expected)
		}
	}
}

func TestScenarioHandlerMapsMissingScenario(t *testing.T) {
	handler, err := NewScenarioHandler(&httpScenarioRepository{})
	if err != nil {
		t.Fatalf("NewScenarioHandler() error = %v", err)
	}
	server := scenarioServer(t, handler)

	response := httptest.NewRecorder()
	server.ServeHTTP(response, adminRequest(http.MethodGet, "/admin/scenarios/missing"))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestScenarioHandlerRejectsMalformedPath(t *testing.T) {
	handler, err := NewScenarioHandler(&httpScenarioRepository{})
	if err != nil {
		t.Fatalf("NewScenarioHandler() error = %v", err)
	}
	server := scenarioServer(t, handler)

	response := httptest.NewRecorder()
	server.ServeHTTP(response, adminRequest(http.MethodGet, "/admin/scenarios/scenario-http/extra"))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestScenarioHandlerMapsRepositoryError(t *testing.T) {
	handler, err := NewScenarioHandler(&httpScenarioRepository{err: errors.New("storage unavailable")})
	if err != nil {
		t.Fatalf("NewScenarioHandler() error = %v", err)
	}
	server := scenarioServer(t, handler)

	response := httptest.NewRecorder()
	server.ServeHTTP(response, adminRequest(http.MethodGet, "/admin/scenarios/scenario-http"))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}

func TestScenarioHandlerCreatesAndExecutesScenario(t *testing.T) {
	repository := &httpScenarioRepository{}
	engine, err := replayapplication.NewReplayEngine(&httpScenarioRunner{result: replayapplication.Result{ScenarioID: "scenario-http", Provider: replaydomain.ProviderConfiguration{ID: "fake"}, CurrentVirtualTime: time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)}})
	if err != nil {
		t.Fatalf("NewReplayEngine() error = %v", err)
	}
	service, err := replayapplication.NewScenarioService(repository, engine)
	if err != nil {
		t.Fatalf("NewScenarioService() error = %v", err)
	}
	handler, err := NewScenarioHandlerWithService(repository, service)
	if err != nil {
		t.Fatalf("NewScenarioHandlerWithService() error = %v", err)
	}
	server := scenarioServer(t, handler)
	body := `{"id":"scenario-http","provider":{"id":"fake"},"initial_virtual_time":"2026-08-26T12:00:00Z","deterministic_configuration":{"seed":42},"initial_payments":[],"commands":[]}`
	create := httptest.NewRecorder()
	request := adminRequest(http.MethodPost, "/admin/scenarios")
	request.Body = io.NopCloser(strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	server.ServeHTTP(create, request)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", create.Code, create.Body.String())
	}
	execute := httptest.NewRecorder()
	server.ServeHTTP(execute, adminRequest(http.MethodPost, "/admin/scenarios/scenario-http/execute"))
	if execute.Code != http.StatusOK || !strings.Contains(execute.Body.String(), `"scenario_id":"scenario-http"`) {
		t.Fatalf("execute = %d: %s", execute.Code, execute.Body.String())
	}
}

type httpScenarioRepository struct {
	scenario *replaydomain.Scenario
	err      error
}

func (r *httpScenarioRepository) Save(_ context.Context, scenario *replaydomain.Scenario) error {
	if r.scenario != nil {
		return replaydomain.ErrScenarioAlreadyExists
	}
	r.scenario = scenario
	return nil
}

func (r *httpScenarioRepository) FindByID(_ context.Context, _ replaydomain.ScenarioID) (*replaydomain.Scenario, error) {
	return r.scenario, r.err
}

type httpScenarioRunner struct{ result replayapplication.Result }

func (r *httpScenarioRunner) Run(context.Context, replaydomain.Scenario) (replayapplication.Result, error) {
	return r.result, nil
}

func scenarioServer(t *testing.T, handler *ScenarioHandler) *api.Server {
	t.Helper()
	server, err := api.NewServer(":8080")
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if err := handler.Register(server, "test-admin-token"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	return server
}

func httpScenario(t *testing.T) *replaydomain.Scenario {
	t.Helper()
	initialAmount, err := paymentdomain.NewMoney(1000, "EUR")
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}
	commandAmount, err := paymentdomain.NewMoney(500, "EUR")
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}
	scenario, err := replaydomain.New(
		"scenario-http",
		[]paymentdomain.PaymentState{{ID: "payment-initial", Amount: initialAmount, Status: paymentdomain.StatusPending, Version: 1}},
		[]replaydomain.Command{{Type: replaydomain.CommandCreatePayment, PaymentID: "payment-created", Amount: commandAmount}},
		replaydomain.ProviderConfiguration{ID: "stripe"},
		time.Date(2026, 2, 3, 4, 5, 6, 0, time.FixedZone("CET", 3600)),
		replaydomain.DeterministicConfiguration{Seed: 99},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return scenario
}

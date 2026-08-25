package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	paymentapplication "proxynth/payment-sandbox/internal/payment/application"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
	providerfake "proxynth/payment-sandbox/internal/provider/fake"
	replaydomain "proxynth/payment-sandbox/internal/replay/domain"
)

func TestRunner_ExecutesCommandsInOrder(t *testing.T) {
	amount := testMoney(t, 10000, "EUR")
	scenario := replaydomain.Scenario{
		ID:                 "scenario-1",
		Provider:           replaydomain.ProviderConfiguration{ID: "fake"},
		InitialVirtualTime: time.Date(2026, 8, 24, 12, 0, 0, 0, time.FixedZone("CET", 3600)),
		DeterministicConfiguration: replaydomain.DeterministicConfiguration{
			Seed: 42,
		},
		Commands: []replaydomain.Command{
			{Type: replaydomain.CommandCreatePayment, PaymentID: "payment-1", Amount: amount},
			{Type: replaydomain.CommandAuthorize, PaymentID: "payment-1"},
			{Type: replaydomain.CommandCapture, PaymentID: "payment-1", Amount: amount},
			{Type: replaydomain.CommandRefund, PaymentID: "payment-1", Amount: amount},
		},
	}

	result, err := testRunner(t).Run(context.Background(), scenario)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.ScenarioID != scenario.ID {
		t.Errorf("ScenarioID = %q, want %q", result.ScenarioID, scenario.ID)
	}
	if result.Provider != scenario.Provider {
		t.Errorf("Provider = %+v, want %+v", result.Provider, scenario.Provider)
	}
	if result.DeterministicConfiguration != scenario.DeterministicConfiguration {
		t.Errorf("DeterministicConfiguration = %+v, want %+v", result.DeterministicConfiguration, scenario.DeterministicConfiguration)
	}
	if !result.CurrentVirtualTime.Equal(scenario.InitialVirtualTime.UTC()) {
		t.Errorf("CurrentVirtualTime = %v, want %v", result.CurrentVirtualTime, scenario.InitialVirtualTime.UTC())
	}

	if len(result.Payments) != 1 {
		t.Fatalf("Payments length = %d, want 1", len(result.Payments))
	}
	if result.Payments[0].Status != paymentdomain.StatusRefunded {
		t.Errorf("payment status = %q, want %q", result.Payments[0].Status, paymentdomain.StatusRefunded)
	}
	if result.Payments[0].Version != 4 {
		t.Errorf("payment version = %d, want 4", result.Payments[0].Version)
	}
}

func TestRunner_ExecutesPaymentSagaThroughSchedulerWorker(t *testing.T) {
	amount := testMoney(t, 10000, "EUR")
	scenario := validScenario(nil, []replaydomain.Command{
		{Type: replaydomain.CommandCreatePayment, PaymentID: "payment-saga", Amount: amount},
		{Type: replaydomain.CommandStartSaga, PaymentID: "payment-saga", Amount: amount},
	})

	result, err := testRunner(t).Run(context.Background(), scenario)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Payments) != 1 || result.Payments[0].Status != paymentdomain.StatusCaptured {
		t.Fatalf("payments = %+v, want captured payment", result.Payments)
	}
}

func TestRunner_RestoresInitialState(t *testing.T) {
	initial := paymentdomain.PaymentState{
		ID:               "payment-1",
		Amount:           testMoney(t, 10000, "EUR"),
		Status:           paymentdomain.StatusAuthorized,
		AuthorizedAmount: 10000,
		Version:          2,
	}
	scenario := validScenario([]paymentdomain.PaymentState{initial}, []replaydomain.Command{
		{Type: replaydomain.CommandCapture, PaymentID: initial.ID, Amount: testMoney(t, 4000, "EUR")},
	})

	result, err := testRunner(t).Run(context.Background(), scenario)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := result.Payments[0]; got.CapturedAmount != 4000 || got.Status != paymentdomain.StatusPartiallyCaptured {
		t.Fatalf("restored payment state = %+v, want partial capture", got)
	}
}

func TestRunner_StopsAtFirstCommandError(t *testing.T) {
	scenario := validScenario(nil, []replaydomain.Command{
		{Type: replaydomain.CommandAuthorize, PaymentID: "missing"},
		{Type: replaydomain.CommandCreatePayment, PaymentID: "should-not-exist", Amount: testMoney(t, 100, "EUR")},
	})

	_, err := testRunner(t).Run(context.Background(), scenario)
	if !errors.Is(err, paymentapplication.ErrPaymentNotFound) {
		t.Fatalf("Run() error = %v, want %v", err, paymentapplication.ErrPaymentNotFound)
	}
}

func TestRunner_RejectsInvalidScenario(t *testing.T) {
	_, err := testRunner(t).Run(context.Background(), replaydomain.Scenario{})
	if !errors.Is(err, replaydomain.ErrInvalidScenarioID) {
		t.Fatalf("Run() error = %v, want %v", err, replaydomain.ErrInvalidScenarioID)
	}
}

func TestRunner_RespectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := testRunner(t).Run(ctx, validScenario(nil, nil))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want %v", err, context.Canceled)
	}
}

func TestRunner_ExecutesProviderOperationsBeforeDomainTransitions(t *testing.T) {
	amount := testMoney(t, 10000, "EUR")
	provider := &recordingProvider{}
	registry := providerdomain.NewRegistry()
	if err := registry.Register(provider); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	runner := NewRunner(registry)
	ctx := context.WithValue(context.Background(), providerContextKey{}, "replay")
	scenario := validScenario(nil, []replaydomain.Command{
		{Type: replaydomain.CommandCreatePayment, PaymentID: "payment-1", Amount: amount},
		{Type: replaydomain.CommandAuthorize, PaymentID: "payment-1"},
		{Type: replaydomain.CommandCapture, PaymentID: "payment-1", Amount: amount},
		{Type: replaydomain.CommandRefund, PaymentID: "payment-1", Amount: amount},
	})

	if _, err := runner.Run(ctx, scenario); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	wantOperations := []string{"authorize", "capture", "refund"}
	if got := provider.operations(); !reflect.DeepEqual(got, wantOperations) {
		t.Fatalf("provider operations = %v, want %v", got, wantOperations)
	}
	if provider.snapshots[0].Status != paymentdomain.StatusPending || provider.snapshots[1].Status != paymentdomain.StatusAuthorized || provider.snapshots[2].Status != paymentdomain.StatusCaptured {
		t.Fatalf("provider snapshots = %+v, want pre-transition states", provider.snapshots)
	}
	for _, got := range provider.contexts {
		if got != "replay" {
			t.Errorf("provider context value = %v, want replay", got)
		}
	}
}

func TestRunner_ResolvesProviderBeforeExecutingCommands(t *testing.T) {
	runner := testRunner(t)
	scenario := validScenario(nil, nil)
	scenario.Provider.ID = "unknown"

	_, err := runner.Run(context.Background(), scenario)
	if !errors.Is(err, providerdomain.ErrProviderNotFound) {
		t.Fatalf("Run() error = %v, want %v", err, providerdomain.ErrProviderNotFound)
	}
}

func TestRunner_PropagatesProviderError(t *testing.T) {
	wantErr := errors.New("provider unavailable")
	provider := &recordingProvider{err: wantErr}
	registry := providerdomain.NewRegistry()
	if err := registry.Register(provider); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	runner := NewRunner(registry)
	scenario := validScenario(nil, []replaydomain.Command{
		{Type: replaydomain.CommandCreatePayment, PaymentID: "payment-1", Amount: testMoney(t, 100, "EUR")},
		{Type: replaydomain.CommandAuthorize, PaymentID: "payment-1"},
	})

	_, err := runner.Run(context.Background(), scenario)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
}

func TestRunner_RejectsInvalidProviderResult(t *testing.T) {
	provider := &recordingProvider{result: providerdomain.OperationResult{Outcome: "unknown"}}
	registry := providerdomain.NewRegistry()
	if err := registry.Register(provider); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	runner := NewRunner(registry)
	scenario := validScenario(nil, []replaydomain.Command{
		{Type: replaydomain.CommandCreatePayment, PaymentID: "payment-1", Amount: testMoney(t, 100, "EUR")},
		{Type: replaydomain.CommandAuthorize, PaymentID: "payment-1"},
	})

	_, err := runner.Run(context.Background(), scenario)
	if !errors.Is(err, providerdomain.ErrInvalidOperationResult) {
		t.Fatalf("Run() error = %v, want %v", err, providerdomain.ErrInvalidOperationResult)
	}
}

func TestRunner_RejectsNilProviderRegistry(t *testing.T) {
	_, err := NewRunner(nil).Run(context.Background(), validScenario(nil, nil))
	if !errors.Is(err, ErrNilProviderRegistry) {
		t.Fatalf("Run() error = %v, want %v", err, ErrNilProviderRegistry)
	}
}

func TestRunnerReturnsProviderAsynchronousOperationsWithVirtualTime(t *testing.T) {
	scheduledAt := time.Date(2026, 8, 24, 12, 5, 0, 0, time.UTC)
	provider := &recordingProvider{result: providerdomain.OperationResult{
		Outcome: providerdomain.OutcomeSucceeded,
		AsyncOperations: []providerdomain.AsyncOperation{{
			ID:          "job-1",
			PaymentID:   "payment-1",
			Type:        "provider.callback",
			Payload:     []byte("payload"),
			ScheduledAt: scheduledAt,
		}},
	}}
	registry := providerdomain.NewRegistry()
	if err := registry.Register(provider); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	runner := NewRunner(registry)
	scenario := validScenario(nil, []replaydomain.Command{
		{Type: replaydomain.CommandCreatePayment, PaymentID: "payment-1", Amount: testMoney(t, 100, "EUR")},
		{Type: replaydomain.CommandAuthorize, PaymentID: "payment-1"},
	})
	result, err := runner.Run(context.Background(), scenario)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(provider.times) != 1 || !provider.times[0].Equal(scenario.InitialVirtualTime) {
		t.Fatalf("provider virtual times = %v, want [%v]", provider.times, scenario.InitialVirtualTime)
	}
	if len(result.AsyncOperations) != 1 || result.AsyncOperations[0].ID != "job-1" {
		t.Fatalf("async operations = %+v, want job-1", result.AsyncOperations)
	}
	provider.result.AsyncOperations[0].Payload[0] = 'X'
	if string(result.AsyncOperations[0].Payload) != "payload" {
		t.Fatalf("result payload = %q, provider mutation leaked across boundary", result.AsyncOperations[0].Payload)
	}
}

func validScenario(
	initialPayments []paymentdomain.PaymentState,
	commands []replaydomain.Command,
) replaydomain.Scenario {
	return replaydomain.Scenario{
		ID:                 "scenario-1",
		InitialPayments:    initialPayments,
		Commands:           commands,
		Provider:           replaydomain.ProviderConfiguration{ID: "fake"},
		InitialVirtualTime: time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC),
	}
}

func testMoney(t *testing.T, amount int64, currency paymentdomain.Currency) paymentdomain.Money {
	t.Helper()

	money, err := paymentdomain.NewMoney(amount, currency)
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}

	return money
}

func testRunner(t *testing.T) *Runner {
	t.Helper()
	registry := providerdomain.NewRegistry()
	if err := registry.Register(providerfake.New()); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	return NewRunner(registry)
}

type providerContextKey struct{}

type recordingProvider struct {
	err       error
	result    providerdomain.OperationResult
	calls     []string
	snapshots []providerdomain.PaymentSnapshot
	contexts  []any
	times     []time.Time
}

func (p *recordingProvider) Identity() providerdomain.ProviderIdentity {
	return providerdomain.ProviderIdentity{ID: "fake"}
}

func (p *recordingProvider) Authorize(ctx context.Context, request providerdomain.AuthorizeRequest) (providerdomain.OperationResult, error) {
	return p.record(ctx, "authorize", request.Payment, request.At)
}

func (p *recordingProvider) Capture(ctx context.Context, request providerdomain.CaptureRequest) (providerdomain.OperationResult, error) {
	return p.record(ctx, "capture", request.Payment, request.At)
}

func (p *recordingProvider) Refund(ctx context.Context, request providerdomain.RefundRequest) (providerdomain.OperationResult, error) {
	return p.record(ctx, "refund", request.Payment, request.At)
}

func (p *recordingProvider) Cancel(ctx context.Context, request providerdomain.CancelRequest) (providerdomain.OperationResult, error) {
	return p.record(ctx, "cancel", request.Payment, request.At)
}

func (p *recordingProvider) record(ctx context.Context, operation string, snapshot providerdomain.PaymentSnapshot, at time.Time) (providerdomain.OperationResult, error) {
	p.calls = append(p.calls, operation)
	p.snapshots = append(p.snapshots, snapshot)
	p.contexts = append(p.contexts, ctx.Value(providerContextKey{}))
	p.times = append(p.times, at)
	if p.err != nil {
		return providerdomain.OperationResult{}, p.err
	}
	if p.result.Outcome == "" {
		return providerdomain.OperationResult{Outcome: providerdomain.OutcomeSucceeded}, nil
	}
	return p.result, nil
}

func (p *recordingProvider) operations() []string {
	return p.calls
}

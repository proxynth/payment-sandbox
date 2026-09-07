package application

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	pd "proxynth/payment-sandbox/internal/payment/domain"
	provider "proxynth/payment-sandbox/internal/provider/domain"
	rd "proxynth/payment-sandbox/internal/replay/domain"
)

// Positive control: simple replay is deterministic across repeated full results.
func TestAuditSimpleReplayDeterminism(t *testing.T) {
	scenario := validScenario(nil, []rd.Command{
		{Type: rd.CommandCreatePayment, PaymentID: "p", Amount: testMoney(t, 1000, "EUR")},
		{Type: rd.CommandAuthorize, PaymentID: "p"},
		{Type: rd.CommandCapture, PaymentID: "p", Amount: testMoney(t, 1000, "EUR")},
	})
	runner := testRunner(t)
	baseline, err := runner.Run(context.Background(), scenario)
	if err != nil {
		t.Fatal(err)
	}
	for range 50 {
		actual, err := runner.Run(context.Background(), scenario)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(baseline, actual) {
			t.Fatal("replay changed")
		}
	}
}

// Invariant: batch selection and final payment results are stable above BatchSize=100.
func TestAuditReplayDeterminismAboveBatchLimit(t *testing.T) {
	scenario := validScenario(nil, nil)
	scenario.Provider.Profile = "pending_authorize"
	scenario.DeterministicConfiguration.Seed = 42
	for i := range 101 {
		id := pd.ID(fmt.Sprintf("p%03d", i))
		scenario.Commands = append(scenario.Commands, rd.Command{Type: rd.CommandCreatePayment, PaymentID: id, Amount: testMoney(t, 1000, "EUR")}, rd.Command{Type: rd.CommandAuthorize, PaymentID: id})
	}
	scenario.Commands = append(scenario.Commands, rd.Command{Type: rd.CommandAdvanceTime, Duration: time.Minute}, rd.Command{Type: rd.CommandExecuteAsync, OperationID: "fake:p000:async"})
	runner := testRunner(t)
	baseline, firstErr := runner.Run(context.Background(), scenario)
	for i := 1; i <= 100; i++ {
		actual, err := runner.Run(context.Background(), scenario)
		if fmt.Sprint(err) != fmt.Sprint(firstErr) {
			t.Fatalf("same scenario+seed: run0 err=%v, run%d err=%v", firstErr, i, err)
		}
		if err == nil && !Compare(baseline, actual).Equivalent {
			t.Fatalf("same scenario+seed: run%d changed payments: %+v", i, Compare(baseline, actual).Differences)
		}
	}
}

// Invariant: comparison must detect a changed future asynchronous effect.
func TestAuditReplayComparisonDetectsAsyncDifference(t *testing.T) {
	baseline := Result{CurrentVirtualTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	actual := baseline
	actual.AsyncOperations = []provider.AsyncOperation{{ID: "extra-job", PaymentID: "p", Type: "authorize", ScheduledAt: baseline.CurrentVirtualTime.Add(time.Hour)}}
	if Compare(baseline, actual).Equivalent {
		t.Fatal("Compare reports equivalent with an extra asynchronous operation")
	}
}

// Invariant: execute_async selects the requested operation, as documented.
func TestAuditExecuteAsyncTargetsOneOperation(t *testing.T) {
	scenario := validScenario(nil, nil)
	scenario.Provider.Profile = "pending_authorize"
	for _, id := range []pd.ID{"a", "b"} {
		scenario.Commands = append(scenario.Commands, rd.Command{Type: rd.CommandCreatePayment, PaymentID: id, Amount: testMoney(t, 1000, "EUR")}, rd.Command{Type: rd.CommandAuthorize, PaymentID: id})
	}
	scenario.Commands = append(scenario.Commands, rd.Command{Type: rd.CommandAdvanceTime, Duration: time.Minute}, rd.Command{Type: rd.CommandExecuteAsync, OperationID: "fake:a:async"})
	result, err := testRunner(t).Run(context.Background(), scenario)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range result.Payments {
		if p.ID == "b" && p.Status != pd.StatusPending {
			t.Fatalf("executing a also mutated b: status=%s", p.Status)
		}
	}
}

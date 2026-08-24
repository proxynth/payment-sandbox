package application

import (
	"context"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/payment/domain"
	"proxynth/payment-sandbox/internal/platform/clock"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
)

func TestDiagnostics_CombinesTimelineRuntimeAndProviders(t *testing.T) {
	payment, err := domain.New("payment-diagnostics", mustMoney(t))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	virtualClock, err := clock.NewVirtualClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewVirtualClock() error = %v", err)
	}
	diagnostics, err := NewDiagnostics(
		&timelinePaymentRepository{payment: payment},
		&timelineEventLog{},
		virtualClock,
		diagnosticsRegistry{ids: []providerdomain.ProviderID{"stripe", "adyen"}},
	)
	if err != nil {
		t.Fatalf("NewDiagnostics() error = %v", err)
	}

	result, err := diagnostics.Execute(context.Background(), payment.ID())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Timeline.Payment != payment {
		t.Fatal("diagnostics did not include the payment")
	}
	if !result.CurrentTime.Equal(virtualClock.Now()) {
		t.Fatalf("CurrentTime = %v, want %v", result.CurrentTime, virtualClock.Now())
	}
	if len(result.ProviderIDs) != 2 || result.ProviderIDs[0] != "adyen" || result.ProviderIDs[1] != "stripe" {
		t.Fatalf("ProviderIDs = %v, want [adyen stripe]", result.ProviderIDs)
	}
}

type diagnosticsRegistry struct {
	ids []providerdomain.ProviderID
}

func (r diagnosticsRegistry) IDs() []providerdomain.ProviderID { return r.ids }

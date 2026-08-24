package application

import (
	"context"
	"errors"
	"sort"
	"time"

	"proxynth/payment-sandbox/internal/payment/application"
	"proxynth/payment-sandbox/internal/payment/domain"
	businessclock "proxynth/payment-sandbox/internal/platform/clock"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
)

var (
	ErrNilDiagnosticsClock    = errors.New("diagnostics clock is nil")
	ErrNilDiagnosticsRegistry = errors.New("diagnostics provider registry is nil")
)

type diagnosticsProviderRegistry interface {
	IDs() []providerdomain.ProviderID
}

type Diagnostics struct {
	timeline  *Timeline
	clock     businessclock.Clock
	providers diagnosticsProviderRegistry
}

type PaymentDiagnostics struct {
	Timeline    PaymentTimeline
	CurrentTime time.Time
	ProviderIDs []providerdomain.ProviderID
}

func NewDiagnostics(
	payments application.Repository,
	events application.EventLog,
	clock businessclock.Clock,
	providers diagnosticsProviderRegistry,
) (*Diagnostics, error) {
	if clock == nil {
		return nil, ErrNilDiagnosticsClock
	}
	if providers == nil {
		return nil, ErrNilDiagnosticsRegistry
	}

	timeline, err := NewTimeline(payments, events)
	if err != nil {
		return nil, err
	}

	return &Diagnostics{timeline: timeline, clock: clock, providers: providers}, nil
}

func (d *Diagnostics) Execute(ctx context.Context, id domain.ID) (PaymentDiagnostics, error) {
	timeline, err := d.timeline.Execute(ctx, id)
	if err != nil {
		return PaymentDiagnostics{}, err
	}

	providerIDs := append([]providerdomain.ProviderID(nil), d.providers.IDs()...)
	sort.Slice(providerIDs, func(i, j int) bool { return providerIDs[i] < providerIDs[j] })

	return PaymentDiagnostics{
		Timeline:    timeline,
		CurrentTime: d.clock.Now().UTC(),
		ProviderIDs: providerIDs,
	}, nil
}

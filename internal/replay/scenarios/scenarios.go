package scenarios

import (
	"errors"
	"time"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
	replaydomain "proxynth/payment-sandbox/internal/replay/domain"
)

type Name string

const (
	PaymentLifecycle    Name = "payment_lifecycle"
	PaymentCancellation Name = "payment_cancellation"
)

var ErrInvalidName = errors.New("invalid provider scenario name")

// Names returns the stable catalogue of canonical provider scenarios.
func Names() []Name {
	return []Name{PaymentLifecycle, PaymentCancellation}
}

// New builds one validated scenario shared by every provider implementation.
func New(
	name Name,
	providerID providerdomain.ProviderID,
	scenarioID replaydomain.ScenarioID,
	initialVirtualTime time.Time,
	amount paymentdomain.Money,
	seed uint64,
) (*replaydomain.Scenario, error) {
	return NewWithProfile(name, providerID, "success", scenarioID, initialVirtualTime, amount, seed)
}

// NewWithProfile builds a scenario with provider-owned deterministic behaviour.
func NewWithProfile(
	name Name,
	providerID providerdomain.ProviderID,
	profile string,
	scenarioID replaydomain.ScenarioID,
	initialVirtualTime time.Time,
	amount paymentdomain.Money,
	seed uint64,
) (*replaydomain.Scenario, error) {
	paymentID := paymentdomain.ID(string(scenarioID) + "-payment")
	commands, err := commandsFor(name, paymentID, amount)
	if err != nil {
		return nil, err
	}

	return replaydomain.New(
		scenarioID,
		nil,
		commands,
		replaydomain.ProviderConfiguration{ID: providerID, Profile: profile},
		initialVirtualTime,
		replaydomain.DeterministicConfiguration{Seed: seed},
	)
}

func commandsFor(name Name, paymentID paymentdomain.ID, amount paymentdomain.Money) ([]replaydomain.Command, error) {
	base := []replaydomain.Command{
		{Type: replaydomain.CommandCreatePayment, PaymentID: paymentID, Amount: amount},
		{Type: replaydomain.CommandAuthorize, PaymentID: paymentID},
	}

	switch name {
	case PaymentLifecycle:
		return append(base,
			replaydomain.Command{Type: replaydomain.CommandCapture, PaymentID: paymentID, Amount: amount},
			replaydomain.Command{Type: replaydomain.CommandRefund, PaymentID: paymentID, Amount: amount},
		), nil
	case PaymentCancellation:
		return append(base, replaydomain.Command{Type: replaydomain.CommandCancel, PaymentID: paymentID}), nil
	default:
		return nil, ErrInvalidName
	}
}

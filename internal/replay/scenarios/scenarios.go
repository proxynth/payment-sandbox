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
	commands, err := commandsFor(name, amount)
	if err != nil {
		return nil, err
	}

	return replaydomain.New(
		scenarioID,
		nil,
		commands,
		replaydomain.ProviderConfiguration{ID: providerID},
		initialVirtualTime,
		replaydomain.DeterministicConfiguration{Seed: seed},
	)
}

func commandsFor(name Name, amount paymentdomain.Money) ([]replaydomain.Command, error) {
	base := []replaydomain.Command{
		{Type: replaydomain.CommandCreatePayment, PaymentID: "payment-1", Amount: amount},
		{Type: replaydomain.CommandAuthorize, PaymentID: "payment-1"},
	}

	switch name {
	case PaymentLifecycle:
		return append(base,
			replaydomain.Command{Type: replaydomain.CommandCapture, PaymentID: "payment-1", Amount: amount},
			replaydomain.Command{Type: replaydomain.CommandRefund, PaymentID: "payment-1", Amount: amount},
		), nil
	case PaymentCancellation:
		return append(base, replaydomain.Command{Type: replaydomain.CommandCancel, PaymentID: "payment-1"}), nil
	default:
		return nil, ErrInvalidName
	}
}

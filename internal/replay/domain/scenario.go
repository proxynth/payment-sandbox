package domain

import (
	"time"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
)

type ScenarioID string

type ProviderConfiguration struct {
	ID      providerdomain.ProviderID
	Profile string
}

func (configuration ProviderConfiguration) Validate() error {
	if err := (providerdomain.ProviderIdentity{ID: configuration.ID}).Validate(); err != nil {
		return ErrInvalidProviderConfiguration
	}

	return nil
}

type DeterministicConfiguration struct {
	Seed uint64
}

// Scenario contains the complete business input required to reproduce a
// deterministic execution. It does not execute commands or persist state.
type Scenario struct {
	ID                         ScenarioID
	InitialPayments            []paymentdomain.PaymentState
	Commands                   []Command
	Provider                   ProviderConfiguration
	InitialVirtualTime         time.Time
	DeterministicConfiguration DeterministicConfiguration
}

func New(
	id ScenarioID,
	initialPayments []paymentdomain.PaymentState,
	commands []Command,
	provider ProviderConfiguration,
	initialVirtualTime time.Time,
	deterministic DeterministicConfiguration,
) (*Scenario, error) {
	scenario := &Scenario{
		ID:                         id,
		InitialPayments:            append([]paymentdomain.PaymentState(nil), initialPayments...),
		Commands:                   append([]Command(nil), commands...),
		Provider:                   provider,
		InitialVirtualTime:         initialVirtualTime.UTC(),
		DeterministicConfiguration: deterministic,
	}

	if err := scenario.Validate(); err != nil {
		return nil, err
	}

	return scenario, nil
}

func (scenario Scenario) Validate() error {
	if scenario.ID == "" {
		return ErrInvalidScenarioID
	}

	if scenario.InitialVirtualTime.IsZero() {
		return ErrInvalidScenarioTime
	}

	if err := scenario.Provider.Validate(); err != nil {
		return err
	}

	seenPayments := make(map[paymentdomain.ID]struct{}, len(scenario.InitialPayments))
	for _, payment := range scenario.InitialPayments {
		if err := payment.Validate(); err != nil {
			return err
		}

		if _, exists := seenPayments[payment.ID]; exists {
			return ErrDuplicateInitialPayment
		}

		seenPayments[payment.ID] = struct{}{}
	}

	for _, command := range scenario.Commands {
		if err := command.Validate(); err != nil {
			return ErrInvalidCommand
		}
	}

	return nil
}

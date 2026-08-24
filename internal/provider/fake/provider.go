package fake

import (
	"proxynth/payment-sandbox/internal/provider/deterministic"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
)

const providerID providerdomain.ProviderID = "fake"

// Provider is the deterministic fake provider used for local simulations and
// contract tests.
type Provider struct {
	*deterministic.Provider
}

var _ providerdomain.Provider = (*Provider)(nil)

func New() *Provider {
	return &Provider{Provider: deterministic.New(providerID)}
}

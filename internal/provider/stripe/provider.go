package stripe

import (
	"proxynth/payment-sandbox/internal/provider/deterministic"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
)

const providerID providerdomain.ProviderID = "stripe"

// Provider is the deterministic Stripe provider implementation. Stripe can
// add provider-specific behaviour here without leaking it into the core.
type Provider struct {
	*deterministic.Provider
}

var _ providerdomain.Provider = (*Provider)(nil)

func New() *Provider {
	return &Provider{Provider: deterministic.New(providerID)}
}

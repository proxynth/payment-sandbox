package adyen

import (
	"proxynth/payment-sandbox/internal/provider/deterministic"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
)

const providerID providerdomain.ProviderID = "adyen"

// Provider is the deterministic Adyen provider implementation. Adyen can add
// provider-specific behaviour here without leaking it into the core.
type Provider struct {
	*deterministic.Provider
}

var _ providerdomain.Provider = (*Provider)(nil)

func New() *Provider {
	return &Provider{Provider: deterministic.New(providerID)}
}

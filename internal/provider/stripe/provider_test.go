package stripe

import (
	"testing"

	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
)

func TestProvider_Identity(t *testing.T) {
	var provider providerdomain.Provider = New()

	if got := provider.Identity().ID; got != providerID {
		t.Fatalf("Identity().ID = %q, want %q", got, providerID)
	}
}

func TestProvider_CanBeRegisteredAndResolved(t *testing.T) {
	registry := providerdomain.NewRegistry()
	provider := New()

	if err := registry.Register(provider); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	resolved, err := registry.Resolve(providerID)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if resolved != provider {
		t.Fatalf("Resolve() returned a different provider")
	}
}

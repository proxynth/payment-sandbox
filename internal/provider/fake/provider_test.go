package fake

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

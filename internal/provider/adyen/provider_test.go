package adyen

import (
	"context"
	"testing"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
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

func TestProvider_ReturnsAdyenReference(t *testing.T) {
	provider := New()
	request := providerdomain.AuthorizeRequest{
		Payment: providerdomain.PaymentSnapshot{
			ID:      "payment-1",
			Amount:  mustMoney(t, 10000, "EUR"),
			Status:  paymentdomain.StatusPending,
			Version: 1,
		},
	}

	result, err := provider.Authorize(context.Background(), request)
	if err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}

	if result.ProviderReference != "adyen:authorize:payment-1" {
		t.Fatalf("ProviderReference = %q, want %q", result.ProviderReference, "adyen:authorize:payment-1")
	}
}

func mustMoney(t *testing.T, amount int64, currency string) paymentdomain.Money {
	t.Helper()

	money, err := paymentdomain.NewMoney(amount, paymentdomain.Currency(currency))
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}

	return money
}

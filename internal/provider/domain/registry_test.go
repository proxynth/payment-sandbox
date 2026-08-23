package domain

import (
	"context"
	"errors"
	"testing"
)

func TestRegistry_RegisterAndResolve(t *testing.T) {
	registry := NewRegistry()
	provider := fakeProvider{}

	if err := registry.Register(provider); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	resolved, err := registry.Resolve("fake")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if resolved != provider {
		t.Fatalf("Resolve() returned a different provider")
	}
}

func TestRegistry_ZeroValueCanRegisterAndResolve(t *testing.T) {
	var registry Registry

	if err := registry.Register(fakeProvider{}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if _, err := registry.Resolve("fake"); err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
}

func TestRegistry_Register_RejectsNilProvider(t *testing.T) {
	registry := NewRegistry()
	var provider *fakeProvider

	err := registry.Register(provider)

	if !errors.Is(err, ErrNilProvider) {
		t.Fatalf("Register() error = %v, want %v", err, ErrNilProvider)
	}
}

func TestRegistry_Register_RejectsInvalidIdentity(t *testing.T) {
	registry := NewRegistry()
	provider := fakeProviderWithID{}

	err := registry.Register(provider)

	if !errors.Is(err, ErrInvalidProviderID) {
		t.Fatalf("Register() error = %v, want %v", err, ErrInvalidProviderID)
	}
}

func TestRegistry_Register_RejectsDuplicateIdentity(t *testing.T) {
	registry := NewRegistry()

	if err := registry.Register(fakeProvider{}); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	err := registry.Register(fakeProvider{})

	if !errors.Is(err, ErrProviderAlreadyExists) {
		t.Fatalf("second Register() error = %v, want %v", err, ErrProviderAlreadyExists)
	}
}

func TestRegistry_Resolve_RejectsUnknownProvider(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Resolve("unknown")

	if !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("Resolve() error = %v, want %v", err, ErrProviderNotFound)
	}
}

func TestRegistry_Resolve_RejectsEmptyProviderID(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Resolve("")

	if !errors.Is(err, ErrInvalidProviderID) {
		t.Fatalf("Resolve() error = %v, want %v", err, ErrInvalidProviderID)
	}
}

func TestRegistry_IDs_ReturnsStableOrder(t *testing.T) {
	registry := NewRegistry()

	for _, provider := range []Provider{
		fakeProviderWithID{identity: ProviderIdentity{ID: "zeta"}},
		fakeProviderWithID{identity: ProviderIdentity{ID: "alpha"}},
	} {
		if err := registry.Register(provider); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
	}

	got := registry.IDs()
	want := []ProviderID{"alpha", "zeta"}

	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("IDs() = %v, want %v", got, want)
	}
}

type fakeProviderWithID struct {
	identity ProviderIdentity
}

func (provider fakeProviderWithID) Identity() ProviderIdentity {
	return provider.identity
}

func (fakeProviderWithID) Authorize(context.Context, AuthorizeRequest) (OperationResult, error) {
	return OperationResult{Outcome: OutcomeSucceeded}, nil
}

func (fakeProviderWithID) Capture(context.Context, CaptureRequest) (OperationResult, error) {
	return OperationResult{Outcome: OutcomeSucceeded}, nil
}

func (fakeProviderWithID) Refund(context.Context, RefundRequest) (OperationResult, error) {
	return OperationResult{Outcome: OutcomeSucceeded}, nil
}

func (fakeProviderWithID) Cancel(context.Context, CancelRequest) (OperationResult, error) {
	return OperationResult{Outcome: OutcomeSucceeded}, nil
}

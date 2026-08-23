package application

import (
	"context"
	"errors"
	"testing"

	"proxynth/payment-sandbox/internal/payment/domain"
)

func TestAuthorizePayment_ExecuteSuccessfully(t *testing.T) {
	payment := newApplicationTestPayment(t)
	repository := &fakeRepository{payment: payment}

	got, err := NewAuthorizePayment(repository).Execute(
		context.Background(),
		AuthorizePaymentCommand{PaymentID: payment.ID()},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got.Status() != domain.StatusAuthorized {
		t.Errorf("Status() = %q, want %q", got.Status(), domain.StatusAuthorized)
	}
	if repository.saveCalls != 1 {
		t.Errorf("Save() calls = %d, want 1", repository.saveCalls)
	}
}

func TestAuthorizePayment_DoesNotPersistInvalidTransition(t *testing.T) {
	payment := newApplicationAuthorizedPayment(t)
	repository := &fakeRepository{payment: payment}

	_, err := NewAuthorizePayment(repository).Execute(
		context.Background(),
		AuthorizePaymentCommand{PaymentID: payment.ID()},
	)

	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidTransition)
	}
	if repository.saveCalls != 0 {
		t.Errorf("Save() calls = %d, want 0", repository.saveCalls)
	}
}

func TestAuthorizePayment_PropagatesRepositoryErrors(t *testing.T) {
	findErr := errors.New("find failed")
	repository := &fakeRepository{findErr: findErr}

	_, err := NewAuthorizePayment(repository).Execute(
		context.Background(),
		AuthorizePaymentCommand{PaymentID: "payment-missing"},
	)

	if !errors.Is(err, findErr) {
		t.Fatalf("Execute() error = %v, want %v", err, findErr)
	}
}

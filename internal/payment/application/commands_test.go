package application

import (
	"context"
	"errors"
	"testing"

	"proxynth/payment-sandbox/internal/payment/domain"
)

func TestCreatePayment_ExecuteSuccessfully(t *testing.T) {
	repository := &fakeRepository{}

	got, err := NewCreatePayment(repository).Execute(
		context.Background(),
		CreatePaymentCommand{ID: "payment-created", Amount: 4999, Currency: "EUR"},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got.Status() != domain.StatusPending {
		t.Errorf("Status() = %q, want %q", got.Status(), domain.StatusPending)
	}
	if repository.saveCalls != 1 {
		t.Errorf("Save() calls = %d, want 1", repository.saveCalls)
	}
}

func TestCreatePayment_DoesNotPersistInvalidPayment(t *testing.T) {
	repository := &fakeRepository{}

	_, err := NewCreatePayment(repository).Execute(
		context.Background(),
		CreatePaymentCommand{ID: "payment-invalid", Amount: 0, Currency: "EUR"},
	)

	if !errors.Is(err, domain.ErrInvalidMoneyAmount) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidMoneyAmount)
	}
	if repository.saveCalls != 0 {
		t.Errorf("Save() calls = %d, want 0", repository.saveCalls)
	}
}

func TestCreatePayment_PropagatesRepositorySaveError(t *testing.T) {
	saveErr := errors.New("save failed")
	repository := &fakeRepository{saveErr: saveErr}

	_, err := NewCreatePayment(repository).Execute(
		context.Background(),
		CreatePaymentCommand{ID: "payment-save-error", Amount: 4999, Currency: "EUR"},
	)

	if !errors.Is(err, saveErr) {
		t.Fatalf("Execute() error = %v, want %v", err, saveErr)
	}
}

func TestAuthorizePayment_PropagatesPaymentNotFound(t *testing.T) {
	repository := &fakeRepository{findErr: ErrPaymentNotFound}

	_, err := NewAuthorizePayment(repository).Execute(
		context.Background(),
		AuthorizePaymentCommand{PaymentID: "payment-missing"},
	)

	if !errors.Is(err, ErrPaymentNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, ErrPaymentNotFound)
	}
}

func TestFailPayment_ExecuteSuccessfully(t *testing.T) {
	payment := newApplicationTestPayment(t)
	repository := &fakeRepository{payment: payment}

	got, err := NewFailPayment(repository).Execute(
		context.Background(),
		FailPaymentCommand{PaymentID: payment.ID()},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got.Status() != domain.StatusFailed {
		t.Errorf("Status() = %q, want %q", got.Status(), domain.StatusFailed)
	}
	if repository.saveCalls != 1 {
		t.Errorf("Save() calls = %d, want 1", repository.saveCalls)
	}
}

func TestCancelPayment_ExecuteSuccessfully(t *testing.T) {
	payment := newApplicationAuthorizedPayment(t)
	repository := &fakeRepository{payment: payment}

	got, err := NewCancelPayment(repository).Execute(
		context.Background(),
		CancelPaymentCommand{PaymentID: payment.ID()},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got.Status() != domain.StatusCancelled {
		t.Errorf("Status() = %q, want %q", got.Status(), domain.StatusCancelled)
	}
	if repository.saveCalls != 1 {
		t.Errorf("Save() calls = %d, want 1", repository.saveCalls)
	}
}

func TestCapturePayment_ExecuteSuccessfully(t *testing.T) {
	payment := newApplicationAuthorizedPayment(t)
	repository := &fakeRepository{payment: payment}

	got, err := NewCapturePayment(repository).Execute(
		context.Background(),
		CapturePaymentCommand{PaymentID: payment.ID(), Amount: 4999, Currency: "EUR"},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got.Status() != domain.StatusCaptured {
		t.Errorf("Status() = %q, want %q", got.Status(), domain.StatusCaptured)
	}
	if repository.saveCalls != 1 {
		t.Errorf("Save() calls = %d, want 1", repository.saveCalls)
	}
}

func TestCapturePayment_DoesNotPersistInvalidAmount(t *testing.T) {
	payment := newApplicationAuthorizedPayment(t)
	repository := &fakeRepository{payment: payment}

	_, err := NewCapturePayment(repository).Execute(
		context.Background(),
		CapturePaymentCommand{PaymentID: payment.ID(), Amount: 5000, Currency: "EUR"},
	)

	if !errors.Is(err, domain.ErrInvalidCapturedAmount) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidCapturedAmount)
	}
	if repository.saveCalls != 0 {
		t.Errorf("Save() calls = %d, want 0", repository.saveCalls)
	}
}

func TestRefundPayment_ExecuteSuccessfully(t *testing.T) {
	payment := newApplicationCapturedPayment(t)
	repository := &fakeRepository{payment: payment}

	got, err := NewRefundPayment(repository).Execute(
		context.Background(),
		RefundPaymentCommand{PaymentID: payment.ID(), Amount: 4999, Currency: "EUR"},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got.Status() != domain.StatusRefunded {
		t.Errorf("Status() = %q, want %q", got.Status(), domain.StatusRefunded)
	}
	if repository.saveCalls != 1 {
		t.Errorf("Save() calls = %d, want 1", repository.saveCalls)
	}
}

func TestRefundPayment_DoesNotPersistInvalidTransition(t *testing.T) {
	payment := newApplicationAuthorizedPayment(t)
	repository := &fakeRepository{payment: payment}

	_, err := NewRefundPayment(repository).Execute(
		context.Background(),
		RefundPaymentCommand{PaymentID: payment.ID(), Amount: 4999, Currency: "EUR"},
	)

	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidTransition)
	}
	if repository.saveCalls != 0 {
		t.Errorf("Save() calls = %d, want 0", repository.saveCalls)
	}
}

func newApplicationTestPayment(t *testing.T) *domain.Payment {
	t.Helper()

	payment, err := domain.New("payment-test", applicationTestMoney(t, 4999, "EUR"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return payment
}

func newApplicationAuthorizedPayment(t *testing.T) *domain.Payment {
	t.Helper()

	payment := newApplicationTestPayment(t)
	if err := payment.Authorize(); err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}

	return payment
}

func newApplicationCapturedPayment(t *testing.T) *domain.Payment {
	t.Helper()

	payment := newApplicationAuthorizedPayment(t)
	if err := payment.Capture(applicationTestMoney(t, 4999, "EUR")); err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	return payment
}

func applicationTestMoney(t *testing.T, amount int64, currency domain.Currency) domain.Money {
	t.Helper()

	money, err := domain.NewMoney(amount, currency)
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}

	return money
}

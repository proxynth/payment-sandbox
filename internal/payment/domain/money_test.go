package domain

import (
	"errors"
	"testing"
)

func TestNewMoney(t *testing.T) {
	money, err := NewMoney(4999, "EUR")
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}

	if money.Amount() != 4999 {
		t.Errorf("Amount() = %d, want 4999", money.Amount())
	}

	if money.Currency() != "EUR" {
		t.Errorf("Currency() = %q, want %q", money.Currency(), "EUR")
	}
}

func TestNewMoney_RejectsNegativeAmount(t *testing.T) {
	_, err := NewMoney(-1, "EUR")

	if !errors.Is(err, ErrInvalidMoneyAmount) {
		t.Fatalf(
			"NewMoney() error = %v, want %v",
			err,
			ErrInvalidMoneyAmount,
		)
	}
}

func TestNewMoney_RejectsInvalidCurrency(t *testing.T) {
	_, err := NewMoney(4999, "EU")

	if !errors.Is(err, ErrInvalidCurrency) {
		t.Fatalf(
			"NewMoney() error = %v, want %v",
			err,
			ErrInvalidCurrency,
		)
	}
}

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

func TestNewMoney_AcceptsZeroAmount(t *testing.T) {
	money, err := NewMoney(0, "EUR")
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}

	if !money.IsZero() {
		t.Errorf("IsZero() = false, want true")
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
	tests := []string{"EU", "EURO", "eur", "E1R", ""}

	for _, currency := range tests {
		t.Run(currency, func(t *testing.T) {
			_, err := NewMoney(4999, Currency(currency))

			if !errors.Is(err, ErrInvalidCurrency) {
				t.Fatalf(
					"NewMoney() error = %v, want %v",
					err,
					ErrInvalidCurrency,
				)
			}
		})
	}
}

func TestMoney_String(t *testing.T) {
	money, err := NewMoney(4999, "EUR")
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}

	if got := money.String(); got != "4999 EUR" {
		t.Errorf("String() = %q, want %q", got, "4999 EUR")
	}
}

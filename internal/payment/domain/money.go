package domain

import "fmt"

type Currency string

type Money struct {
	amount   int64
	currency Currency
}

func NewMoney(amount int64, currency Currency) (Money, error) {
	if amount < 0 {
		return Money{}, ErrInvalidMoneyAmount
	}

	if !validCurrency(currency) {
		return Money{}, ErrInvalidCurrency
	}

	return Money{amount: amount, currency: currency}, nil
}

func validCurrency(currency Currency) bool {
	if len(currency) != 3 {
		return false
	}

	for _, character := range currency {
		if character < 'A' || character > 'Z' {
			return false
		}
	}

	return true
}

func (m Money) Amount() int64 {
	return m.amount
}

func (m Money) Currency() Currency {
	return m.currency
}

func (m Money) IsZero() bool {
	return m.amount == 0
}

func (m Money) String() string {
	return fmt.Sprintf("%d %s", m.amount, m.currency)
}

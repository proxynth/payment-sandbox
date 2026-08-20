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

	if len(currency) != 3 {
		return Money{}, ErrInvalidCurrency
	}

	return Money{amount: amount, currency: currency}, nil
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

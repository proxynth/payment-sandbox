package domain

// EventSnapshot is the complete payment state carried by each new business
// event. It makes the event log independently verifiable without retaining a
// mutable reference to the current-state table.
type EventSnapshot struct {
	PaymentID        ID       `json:"payment_id"`
	Amount           int64    `json:"amount"`
	Currency         Currency `json:"currency"`
	AuthorizedAmount int64    `json:"authorized_amount"`
	CapturedAmount   int64    `json:"captured_amount"`
	RefundedAmount   int64    `json:"refunded_amount"`
	Status           Status   `json:"status"`
	Version          uint64   `json:"version"`
}

// NewEventSnapshot captures a payment's complete current state for an event.
func NewEventSnapshot(payment *Payment) EventSnapshot {
	return EventSnapshot{
		PaymentID: payment.ID(), Amount: payment.Amount().Amount(), Currency: payment.Amount().Currency(),
		AuthorizedAmount: payment.AuthorizedAmount().Amount(), CapturedAmount: payment.CapturedAmount().Amount(),
		RefundedAmount: payment.RefundedAmount().Amount(), Status: payment.Status(), Version: payment.Version(),
	}
}

// Restore validates and reconstructs a payment from the immutable snapshot.
func (s EventSnapshot) Restore() (*Payment, error) {
	money, err := NewMoney(s.Amount, s.Currency)
	if err != nil {
		return nil, err
	}
	return Restore(PaymentState{
		ID: s.PaymentID, Amount: money, Status: s.Status, AuthorizedAmount: s.AuthorizedAmount,
		CapturedAmount: s.CapturedAmount, RefundedAmount: s.RefundedAmount, Version: s.Version,
	})
}

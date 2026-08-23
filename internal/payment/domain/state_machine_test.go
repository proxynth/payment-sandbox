package domain

import (
	"errors"
	"testing"
)

func TestPayment_TerminalStatesRejectEveryLifecycleOperation(t *testing.T) {
	tests := []struct {
		name  string
		state Status
	}{
		{name: "failed", state: StatusFailed},
		{name: "cancelled", state: StatusCancelled},
		{name: "refunded", state: StatusRefunded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, err := Restore(mustPaymentState(t, tt.state))
			if err != nil {
				t.Fatalf("Restore() error = %v", err)
			}

			beforeVersion := payment.Version()
			beforeStatus := payment.Status()
			beforeAuthorized := payment.AuthorizedAmount()
			beforeCaptured := payment.CapturedAmount()
			beforeRefunded := payment.RefundedAmount()

			operations := []struct {
				name string
				fn   func() error
			}{
				{name: "authorize", fn: payment.Authorize},
				{name: "fail", fn: payment.Fail},
				{name: "cancel", fn: payment.Cancel},
				{name: "capture", fn: func() error {
					return payment.Capture(mustMoney(t, 1000, "EUR"))
				}},
				{name: "refund", fn: func() error {
					return payment.Refund(mustMoney(t, 1000, "EUR"))
				}},
			}

			for _, operation := range operations {
				t.Run(operation.name, func(t *testing.T) {
					err := operation.fn()
					if !errors.Is(err, ErrInvalidTransition) {
						t.Fatalf("operation error = %v, want %v", err, ErrInvalidTransition)
					}

					if payment.Version() != beforeVersion {
						t.Errorf("version changed after rejected operation")
					}
					if payment.Status() != beforeStatus {
						t.Errorf("status changed after rejected operation")
					}
					if payment.AuthorizedAmount() != beforeAuthorized {
						t.Errorf("authorized amount changed after rejected operation")
					}
					if payment.CapturedAmount() != beforeCaptured {
						t.Errorf("captured amount changed after rejected operation")
					}
					if payment.RefundedAmount() != beforeRefunded {
						t.Errorf("refunded amount changed after rejected operation")
					}
				})
			}
		})
	}
}

func TestPaymentState_RejectsInconsistentLifecycleCombinations(t *testing.T) {
	amount := mustMoney(t, 10000, "EUR")

	tests := []struct {
		name  string
		state PaymentState
	}{
		{
			name: "pending with authorized amount",
			state: PaymentState{
				ID: "payment-pending-invalid", Amount: amount, Status: StatusPending,
				AuthorizedAmount: 1, Version: 1,
			},
		},
		{
			name: "authorized without full authorization",
			state: PaymentState{
				ID: "payment-authorized-invalid", Amount: amount, Status: StatusAuthorized,
				AuthorizedAmount: 0, Version: 1,
			},
		},
		{
			name: "partial capture at full amount",
			state: PaymentState{
				ID: "payment-partial-capture-invalid", Amount: amount, Status: StatusPartiallyCaptured,
				AuthorizedAmount: 10000, CapturedAmount: 10000, Version: 1,
			},
		},
		{
			name: "captured before full capture",
			state: PaymentState{
				ID: "payment-captured-invalid", Amount: amount, Status: StatusCaptured,
				AuthorizedAmount: 10000, CapturedAmount: 5000, Version: 1,
			},
		},
		{
			name: "partial refund at full refund",
			state: PaymentState{
				ID: "payment-partial-refund-invalid", Amount: amount, Status: StatusPartiallyRefunded,
				AuthorizedAmount: 10000, CapturedAmount: 10000, RefundedAmount: 10000, Version: 1,
			},
		},
		{
			name: "refunded before full refund",
			state: PaymentState{
				ID: "payment-refunded-invalid", Amount: amount, Status: StatusRefunded,
				AuthorizedAmount: 10000, CapturedAmount: 10000, RefundedAmount: 5000, Version: 1,
			},
		},
		{
			name: "cancelled after capture",
			state: PaymentState{
				ID: "payment-cancelled-invalid", Amount: amount, Status: StatusCancelled,
				AuthorizedAmount: 10000, CapturedAmount: 1, Version: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Restore(tt.state)
			if !errors.Is(err, ErrInvalidPaymentState) {
				t.Fatalf("Restore() error = %v, want %v", err, ErrInvalidPaymentState)
			}
		})
	}
}

package domain

import (
	"errors"
	"testing"
)

func newTestAmount(t *testing.T, amount int64, currency Currency) Money {
	moneyAmount, err := NewMoney(amount, currency)

	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}

	return moneyAmount
}

func TestNew(t *testing.T) {
	payment, err := New("payment-1", newTestAmount(t, 2999, "EUR"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if payment.ID() != "payment-1" {
		t.Fatalf("ID() = %q, want %q", payment.ID(), "payment-1")
	}

	if payment.Status() != StatusPending {
		t.Fatalf("Status() = %q, want %q", payment.Status(), StatusPending)
	}

	if payment.Version() != 1 {
		t.Fatalf("Version() = %d, want %d", payment.Version(), 1)
	}
}

func TestNew_RejectsEmptyID(t *testing.T) {
	_, err := New("", newTestAmount(t, 2999, "EUR"))

	if !errors.Is(err, ErrInvalidPaymentID) {
		t.Fatalf("New() error = %v, want %v", err, ErrInvalidPaymentID)
	}
}

func TestRestore(t *testing.T) {
	money := mustMoney(t, 10000, "EUR")
	payment, err := Restore(PaymentState{
		ID:               "payment-1",
		Amount:           money,
		Status:           StatusCaptured,
		AuthorizedAmount: 10000,
		CapturedAmount:   10000,
		RefundedAmount:   0,
		Version:          4,
	})

	if err != nil {
		t.Fatalf("failed to restore payment: %v", err)
	}

	if payment.ID() != "payment-1" {
		t.Fatalf("payment id should be 'payment-1'")
	}

	if payment.Status() != StatusCaptured {
		t.Fatalf("payment status should be '%s'", StatusCaptured)
	}

	if payment.Version() != 4 {
		t.Fatalf("payment version should be 4")
	}
}

func TestRestore_RejectsEmptyID(t *testing.T) {
	money := mustMoney(t, 10000, "EUR")
	_, err := Restore(PaymentState{
		ID:      "",
		Amount:  money,
		Status:  StatusPending,
		Version: 1,
	})

	if !errors.Is(err, ErrInvalidPaymentID) {
		t.Fatalf("error should be 'ErrInvalidPaymentID'")
	}
}

func TestRestore_RejectsInvalidStatus(t *testing.T) {
	money := mustMoney(t, 10000, "EUR")
	_, err := Restore(PaymentState{
		ID:               "payment-test",
		Amount:           money,
		Status:           Status("foobar"),
		AuthorizedAmount: 10000,
		CapturedAmount:   10000,
		RefundedAmount:   0,
		Version:          1,
	})

	if !errors.Is(err, ErrInvalidPaymentStatus) {
		t.Fatalf("error should be 'ErrInvalidPaymentStatus'")
	}
}

func TestRestore_RejectsInvalidVersion(t *testing.T) {
	_, err := Restore(PaymentState{
		ID:               "payment-test",
		Amount:           mustMoney(t, 10000, "EUR"),
		Status:           StatusPending,
		AuthorizedAmount: 10000,
		CapturedAmount:   10000,
		RefundedAmount:   0,
		Version:          0,
	})

	if !errors.Is(err, ErrInvalidPaymentVersion) {
		t.Fatalf("error should be 'ErrInvalidPaymentVersion'")
	}
}

func TestStatus_Valid(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{
			name:   "pending",
			status: StatusPending,
			want:   true,
		},
		{
			name:   "unknown",
			status: Status("foobar"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.Valid(); got != tt.want {
				t.Errorf("Status.Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPayment_Authorize(t *testing.T) {
	payment := newPendingPayment(t, 10000, "EUR")

	if err := payment.Authorize(); err != nil {
		t.Fatalf("Payment.Authorize() error = %v", err)
	}

	if payment.Status() != StatusAuthorized {
		t.Errorf("payment.Status() = %s, want %s", payment.Status(), StatusAuthorized)
	}

	if payment.AuthorizedAmount().Amount() != 10000 {
		t.Errorf("payment.authorizedAmount() = %d, want %d", payment.AuthorizedAmount().Amount(), 10000)
	}

	if payment.Version() != 2 {
		t.Errorf("payment.Version() = %d, want %d", payment.Version(), 2)
	}
}

func TestPayment_PartialCapture(t *testing.T) {
	payment := newAuthorizedPayment(t, 10000, "EUR")

	amount, err := NewMoney(4000, "EUR")
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}

	if err := payment.Capture(amount); err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	if payment.Status() != StatusPartiallyCaptured {
		t.Errorf(
			"Status() = %q, want %q",
			payment.Status(),
			StatusPartiallyCaptured,
		)
	}

	if payment.CapturedAmount().Amount() != 4000 {
		t.Errorf(
			"CapturedAmount() = %d, want 4000",
			payment.CapturedAmount().Amount(),
		)
	}

	if payment.RemainingCapturableAmount().Amount() != 6000 {
		t.Errorf(
			"RemainingCapturableAmount() = %d, want 6000",
			payment.RemainingCapturableAmount().Amount(),
		)
	}
}

func TestPayment_FullCaptureAfterPartialCapture(t *testing.T) {
	payment := newAuthorizedPayment(t, 10000, "EUR")

	first := mustMoney(t, 4000, "EUR")
	second := mustMoney(t, 6000, "EUR")

	if err := payment.Capture(first); err != nil {
		t.Fatalf("first Capture() error = %v", err)
	}

	if err := payment.Capture(second); err != nil {
		t.Fatalf("second Capture() error = %v", err)
	}

	if payment.Status() != StatusCaptured {
		t.Errorf(
			"Status() = %q, want %q",
			payment.Status(),
			StatusCaptured,
		)
	}

	if payment.CapturedAmount().Amount() != 10000 {
		t.Errorf(
			"CapturedAmount() = %d, want 10000",
			payment.CapturedAmount().Amount(),
		)
	}
}

func TestPayment_CaptureRejectsAmountAboveRemainingCapturable(t *testing.T) {
	payment := newAuthorizedPayment(t, 10000, "EUR")

	if err := payment.Capture(
		mustMoney(t, 6000, "EUR"),
	); err != nil {
		t.Fatalf("first Capture() error = %v", err)
	}

	beforeVersion := payment.Version()
	beforeCaptured := payment.CapturedAmount()
	beforeStatus := payment.Status()

	err := payment.Capture(
		mustMoney(t, 5000, "EUR"),
	)

	if !errors.Is(err, ErrInvalidCapturedAmount) {
		t.Fatalf(
			"Capture() error = %v, want %v",
			err,
			ErrInvalidCapturedAmount,
		)
	}

	if payment.Version() != beforeVersion {
		t.Errorf("version changed after rejected capture")
	}

	if payment.CapturedAmount() != beforeCaptured {
		t.Errorf("captured amount changed after rejected capture")
	}

	if payment.Status() != beforeStatus {
		t.Errorf("status changed after rejected capture")
	}
}

func TestRestore_RejectsCapturedAmountAboveAuthorizedAmount(t *testing.T) {
	money := mustMoney(t, 10000, "EUR")

	_, err := Restore(PaymentState{
		ID:               "payment-1",
		Amount:           money,
		Status:           StatusCaptured,
		AuthorizedAmount: 10000,
		CapturedAmount:   11000,
		RefundedAmount:   0,
		Version:          3,
	})

	if !errors.Is(err, ErrInvalidCapturedAmount) {
		t.Fatalf(
			"Restore() error = %v, want %v",
			err,
			ErrInvalidCapturedAmount,
		)
	}
}

func TestRestore_RejectsRefundedAmountAboveCapturedAmount(t *testing.T) {
	money := mustMoney(t, 10000, "EUR")

	_, err := Restore(PaymentState{
		ID:               "payment-1",
		Amount:           money,
		Status:           StatusRefunded,
		AuthorizedAmount: 10000,
		CapturedAmount:   10000,
		RefundedAmount:   11000,
		Version:          4,
	})

	if !errors.Is(err, ErrInvalidRefundedAmount) {
		t.Fatalf(
			"Restore() error = %v, want %v",
			err,
			ErrInvalidRefundedAmount,
		)
	}
}

func TestPayment_CaptureRejectsInvalidStates(t *testing.T) {
	tests := []struct {
		name   string
		status Status
	}{
		{"pending", StatusPending},
		{"failed", StatusFailed},
		{"cancelled", StatusCancelled},
		{"captured", StatusCaptured},
		{"refunded", StatusRefunded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, err := Restore(mustPaymentState(t, tt.status))
			if err != nil {
				t.Fatalf("Restore() error = %v", err)
			}

			beforeVersion := payment.Version()
			beforeStatus := payment.Status()
			beforeCaptured := payment.CapturedAmount()

			err = payment.Capture(mustMoney(t, 1000, "EUR"))

			if !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf(
					"Capture() error = %v, want %v",
					err,
					ErrInvalidTransition,
				)
			}

			if payment.Version() != beforeVersion {
				t.Errorf("version changed after rejected transition")
			}

			if payment.Status() != beforeStatus {
				t.Errorf("status changed after rejected transition")
			}

			if payment.CapturedAmount() != beforeCaptured {
				t.Errorf("captured amount changed after rejected transition")
			}
		})
	}
}

func TestPayment_Fail(t *testing.T) {
	payment := newPendingPayment(t, 10000, "EUR")

	if err := payment.Fail(); err != nil {
		t.Fatalf("Fail() error = %v", err)
	}

	if payment.Status() != StatusFailed {
		t.Errorf(
			"Status() = %q, want %q",
			payment.Status(),
			StatusFailed,
		)
	}

	if payment.Version() != 2 {
		t.Errorf(
			"Version() = %d, want %d",
			payment.Version(),
			2,
		)
	}

	if payment.AuthorizedAmount().Amount() != 0 {
		t.Errorf(
			"AuthorizedAmount() = %d, want %d",
			payment.AuthorizedAmount().Amount(),
			0,
		)
	}

	if payment.CapturedAmount().Amount() != 0 {
		t.Errorf(
			"CapturedAmount() = %d, want %d",
			payment.CapturedAmount().Amount(),
			0,
		)
	}

	if payment.RefundedAmount().Amount() != 0 {
		t.Errorf(
			"RefundedAmount() = %d, want %d",
			payment.RefundedAmount().Amount(),
			0,
		)
	}
}

func TestPayment_Cancel(t *testing.T) {
	payment := newAuthorizedPayment(t, 10000, "EUR")

	if err := payment.Cancel(); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}

	if payment.Status() != StatusCancelled {
		t.Errorf(
			"Status() = %q, want %q",
			payment.Status(),
			StatusCancelled,
		)
	}

	if payment.AuthorizedAmount().Amount() != 10000 {
		t.Errorf(
			"AuthorizedAmount() = %d, want 10000",
			payment.AuthorizedAmount().Amount(),
		)
	}

	if payment.CapturedAmount().Amount() != 0 {
		t.Errorf(
			"CapturedAmount() = %d, want 0",
			payment.CapturedAmount().Amount(),
		)
	}

	if payment.Version() != 3 {
		t.Errorf(
			"Version() = %d, want 3",
			payment.Version(),
		)
	}
}

func TestPayment_FullCapture(t *testing.T) {
	payment := newAuthorizedPayment(t, 10000, "EUR")

	if err := payment.Capture(
		mustMoney(t, 10000, "EUR"),
	); err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	if payment.Status() != StatusCaptured {
		t.Errorf(
			"Status() = %q, want %q",
			payment.Status(),
			StatusCaptured,
		)
	}

	if payment.CapturedAmount().Amount() != 10000 {
		t.Errorf(
			"CapturedAmount() = %d, want 10000",
			payment.CapturedAmount().Amount(),
		)
	}

	if payment.RemainingCapturableAmount().Amount() != 0 {
		t.Errorf(
			"RemainingCapturableAmount() = %d, want 0",
			payment.RemainingCapturableAmount().Amount(),
		)
	}

	if payment.Version() != 3 {
		t.Errorf(
			"Version() = %d, want 3",
			payment.Version(),
		)
	}
}

func TestPayment_PartialRefund(t *testing.T) {
	payment := newAuthorizedPayment(t, 10000, "EUR")

	if err := payment.Capture(
		mustMoney(t, 10000, "EUR"),
	); err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	if err := payment.Refund(
		mustMoney(t, 3000, "EUR"),
	); err != nil {
		t.Fatalf("Refund() error = %v", err)
	}

	if payment.Status() != StatusPartiallyRefunded {
		t.Errorf(
			"Status() = %q, want %q",
			payment.Status(),
			StatusPartiallyRefunded,
		)
	}

	if payment.CapturedAmount().Amount() != 10000 {
		t.Errorf(
			"CapturedAmount() = %d, want 10000",
			payment.CapturedAmount().Amount(),
		)
	}

	if payment.RefundedAmount().Amount() != 3000 {
		t.Errorf(
			"RefundedAmount() = %d, want 3000",
			payment.RefundedAmount().Amount(),
		)
	}

	if payment.RemainingRefundableAmount().Amount() != 7000 {
		t.Errorf(
			"RemainingRefundableAmount() = %d, want 7000",
			payment.RemainingRefundableAmount().Amount(),
		)
	}

	if payment.Version() != 4 {
		t.Errorf(
			"Version() = %d, want 4",
			payment.Version(),
		)
	}
}

func TestPayment_FullRefund(t *testing.T) {
	payment := newAuthorizedPayment(t, 10000, "EUR")

	if err := payment.Capture(
		mustMoney(t, 10000, "EUR"),
	); err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	if err := payment.Refund(
		mustMoney(t, 10000, "EUR"),
	); err != nil {
		t.Fatalf("Refund() error = %v", err)
	}

	if payment.Status() != StatusRefunded {
		t.Errorf(
			"Status() = %q, want %q",
			payment.Status(),
			StatusRefunded,
		)
	}

	if payment.RefundedAmount().Amount() != 10000 {
		t.Errorf(
			"RefundedAmount() = %d, want 10000",
			payment.RefundedAmount().Amount(),
		)
	}

	if payment.RemainingRefundableAmount().Amount() != 0 {
		t.Errorf(
			"RemainingRefundableAmount() = %d, want 0",
			payment.RemainingRefundableAmount().Amount(),
		)
	}

	if payment.Version() != 4 {
		t.Errorf(
			"Version() = %d, want 4",
			payment.Version(),
		)
	}
}

func TestPayment_FullRefundAfterPartialRefund(t *testing.T) {
	payment := newAuthorizedPayment(t, 10000, "EUR")

	if err := payment.Capture(
		mustMoney(t, 10000, "EUR"),
	); err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	if err := payment.Refund(
		mustMoney(t, 3000, "EUR"),
	); err != nil {
		t.Fatalf("first Refund() error = %v", err)
	}

	if payment.Status() != StatusPartiallyRefunded {
		t.Fatalf(
			"Status() = %q after first refund, want %q",
			payment.Status(),
			StatusPartiallyRefunded,
		)
	}

	if err := payment.Refund(
		mustMoney(t, 7000, "EUR"),
	); err != nil {
		t.Fatalf("second Refund() error = %v", err)
	}

	if payment.Status() != StatusRefunded {
		t.Errorf(
			"Status() = %q, want %q",
			payment.Status(),
			StatusRefunded,
		)
	}

	if payment.RefundedAmount().Amount() != 10000 {
		t.Errorf(
			"RefundedAmount() = %d, want 10000",
			payment.RefundedAmount().Amount(),
		)
	}

	if payment.RemainingRefundableAmount().Amount() != 0 {
		t.Errorf(
			"RemainingRefundableAmount() = %d, want 0",
			payment.RemainingRefundableAmount().Amount(),
		)
	}

	if payment.Version() != 5 {
		t.Errorf(
			"Version() = %d, want 5",
			payment.Version(),
		)
	}
}

func TestPayment_AuthorizeRejectsInvalidStates(t *testing.T) {
	tests := []struct {
		name   string
		status Status
	}{
		{"authorized", StatusAuthorized},
		{"partially captured", StatusPartiallyCaptured},
		{"captured", StatusCaptured},
		{"partially refunded", StatusPartiallyRefunded},
		{"refunded", StatusRefunded},
		{"failed", StatusFailed},
		{"cancelled", StatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, err := Restore(mustPaymentState(t, tt.status))
			if err != nil {
				t.Fatalf("Restore() error = %v", err)
			}

			beforeStatus := payment.Status()
			beforeVersion := payment.Version()
			beforeAuthorized := payment.AuthorizedAmount()

			err = payment.Authorize()

			if !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf(
					"Authorize() error = %v, want %v",
					err,
					ErrInvalidTransition,
				)
			}

			if payment.Status() != beforeStatus {
				t.Errorf("status changed after rejected transition")
			}

			if payment.Version() != beforeVersion {
				t.Errorf("version changed after rejected transition")
			}

			if payment.AuthorizedAmount() != beforeAuthorized {
				t.Errorf("authorized amount changed after rejected transition")
			}
		})
	}
}

func TestPayment_CancelRejectsInvalidStates(t *testing.T) {
	tests := []struct {
		name   string
		status Status
	}{
		{"pending", StatusPending},
		{"partially captured", StatusPartiallyCaptured},
		{"captured", StatusCaptured},
		{"partially refunded", StatusPartiallyRefunded},
		{"refunded", StatusRefunded},
		{"failed", StatusFailed},
		{"cancelled", StatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, err := Restore(mustPaymentState(t, tt.status))
			if err != nil {
				t.Fatalf("Restore() error = %v", err)
			}

			beforeStatus := payment.Status()
			beforeVersion := payment.Version()
			beforeCaptured := payment.CapturedAmount()

			err = payment.Cancel()

			if !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf(
					"Cancel() error = %v, want %v",
					err,
					ErrInvalidTransition,
				)
			}

			if payment.Status() != beforeStatus {
				t.Errorf("status changed after rejected transition")
			}

			if payment.Version() != beforeVersion {
				t.Errorf("version changed after rejected transition")
			}

			if payment.CapturedAmount() != beforeCaptured {
				t.Errorf("captured amount changed after rejected transition")
			}
		})
	}
}

func TestPayment_RefundRejectsInvalidStates(t *testing.T) {
	tests := []struct {
		name   string
		status Status
	}{
		{"pending", StatusPending},
		{"authorized", StatusAuthorized},
		{"partially captured", StatusPartiallyCaptured},
		{"refunded", StatusRefunded},
		{"failed", StatusFailed},
		{"cancelled", StatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, err := Restore(mustPaymentState(t, tt.status))
			if err != nil {
				t.Fatalf("Restore() error = %v", err)
			}

			beforeStatus := payment.Status()
			beforeVersion := payment.Version()
			beforeRefunded := payment.RefundedAmount()

			err = payment.Refund(mustMoney(t, 1000, "EUR"))

			if !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf(
					"Refund() error = %v, want %v",
					err,
					ErrInvalidTransition,
				)
			}

			if payment.Status() != beforeStatus {
				t.Errorf("status changed after rejected transition")
			}

			if payment.Version() != beforeVersion {
				t.Errorf("version changed after rejected transition")
			}

			if payment.RefundedAmount() != beforeRefunded {
				t.Errorf("refunded amount changed after rejected transition")
			}
		})
	}
}

func TestPayment_CaptureRejectsZeroAmount(t *testing.T) {
	payment := newAuthorizedPayment(t, 10000, "EUR")

	beforeStatus := payment.Status()
	beforeVersion := payment.Version()
	beforeCaptured := payment.CapturedAmount()

	err := payment.Capture(mustMoney(t, 0, "EUR"))

	if !errors.Is(err, ErrInvalidCapturedAmount) {
		t.Fatalf(
			"Capture() error = %v, want %v",
			err,
			ErrInvalidCapturedAmount,
		)
	}

	if payment.Status() != beforeStatus {
		t.Errorf("status changed after rejected capture")
	}

	if payment.Version() != beforeVersion {
		t.Errorf("version changed after rejected capture")
	}

	if payment.CapturedAmount() != beforeCaptured {
		t.Errorf("captured amount changed after rejected capture")
	}
}

func TestPayment_CaptureRejectsDifferentCurrency(t *testing.T) {
	payment := newAuthorizedPayment(t, 10000, "EUR")

	beforeStatus := payment.Status()
	beforeVersion := payment.Version()
	beforeCaptured := payment.CapturedAmount()

	err := payment.Capture(mustMoney(t, 1000, "USD"))

	if !errors.Is(err, ErrInvalidCurrency) {
		t.Fatalf(
			"Capture() error = %v, want %v",
			err,
			ErrInvalidCurrency,
		)
	}

	if payment.Status() != beforeStatus {
		t.Errorf("status changed after rejected capture")
	}

	if payment.Version() != beforeVersion {
		t.Errorf("version changed after rejected capture")
	}

	if payment.CapturedAmount() != beforeCaptured {
		t.Errorf("captured amount changed after rejected capture")
	}
}

func TestPayment_RefundRejectsZeroAmount(t *testing.T) {
	payment := newAuthorizedPayment(t, 10000, "EUR")

	if err := payment.Capture(
		mustMoney(t, 10000, "EUR"),
	); err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	beforeStatus := payment.Status()
	beforeVersion := payment.Version()
	beforeRefunded := payment.RefundedAmount()

	err := payment.Refund(mustMoney(t, 0, "EUR"))

	if !errors.Is(err, ErrInvalidRefundedAmount) {
		t.Fatalf(
			"Refund() error = %v, want %v",
			err,
			ErrInvalidRefundedAmount,
		)
	}

	if payment.Status() != beforeStatus {
		t.Errorf("status changed after rejected refund")
	}

	if payment.Version() != beforeVersion {
		t.Errorf("version changed after rejected refund")
	}

	if payment.RefundedAmount() != beforeRefunded {
		t.Errorf("refunded amount changed after rejected refund")
	}
}

func TestPayment_RefundRejectsAmountAboveRemainingRefundable(t *testing.T) {
	payment := newAuthorizedPayment(t, 10000, "EUR")

	if err := payment.Capture(
		mustMoney(t, 10000, "EUR"),
	); err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	if err := payment.Refund(
		mustMoney(t, 7000, "EUR"),
	); err != nil {
		t.Fatalf("first Refund() error = %v", err)
	}

	beforeStatus := payment.Status()
	beforeVersion := payment.Version()
	beforeRefunded := payment.RefundedAmount()

	err := payment.Refund(mustMoney(t, 4000, "EUR"))

	if !errors.Is(err, ErrInvalidRefundedAmount) {
		t.Fatalf(
			"Refund() error = %v, want %v",
			err,
			ErrInvalidRefundedAmount,
		)
	}

	if payment.Status() != beforeStatus {
		t.Errorf("status changed after rejected refund")
	}

	if payment.Version() != beforeVersion {
		t.Errorf("version changed after rejected refund")
	}

	if payment.RefundedAmount() != beforeRefunded {
		t.Errorf("refunded amount changed after rejected refund")
	}
}

func TestPayment_RefundRejectsDifferentCurrency(t *testing.T) {
	payment := newAuthorizedPayment(t, 10000, "EUR")

	if err := payment.Capture(
		mustMoney(t, 10000, "EUR"),
	); err != nil {
		t.Fatalf("Capture() error = %v", err)
	}

	beforeStatus := payment.Status()
	beforeVersion := payment.Version()
	beforeRefunded := payment.RefundedAmount()

	err := payment.Refund(mustMoney(t, 1000, "USD"))

	if !errors.Is(err, ErrInvalidCurrency) {
		t.Fatalf(
			"Refund() error = %v, want %v",
			err,
			ErrInvalidCurrency,
		)
	}

	if payment.Status() != beforeStatus {
		t.Errorf("status changed after rejected refund")
	}

	if payment.Version() != beforeVersion {
		t.Errorf("version changed after rejected refund")
	}

	if payment.RefundedAmount() != beforeRefunded {
		t.Errorf("refunded amount changed after rejected refund")
	}
}

func TestPayment_FailRejectsInvalidStates(t *testing.T) {
	tests := []struct {
		name   string
		status Status
	}{
		{"authorized", StatusAuthorized},
		{"partially captured", StatusPartiallyCaptured},
		{"captured", StatusCaptured},
		{"partially refunded", StatusPartiallyRefunded},
		{"refunded", StatusRefunded},
		{"failed", StatusFailed},
		{"cancelled", StatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, err := Restore(mustPaymentState(t, tt.status))
			if err != nil {
				t.Fatalf("Restore() error = %v", err)
			}

			beforeStatus := payment.Status()
			beforeVersion := payment.Version()

			err = payment.Fail()

			if !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf(
					"Fail() error = %v, want %v",
					err,
					ErrInvalidTransition,
				)
			}

			if payment.Status() != beforeStatus {
				t.Errorf("status changed after rejected transition")
			}

			if payment.Version() != beforeVersion {
				t.Errorf("version changed after rejected transition")
			}
		})
	}
}

func newPendingPayment(
	t *testing.T,
	amount int64,
	currency Currency,
) *Payment {
	t.Helper()

	payment, err := New(
		"payment-test",
		mustMoney(t, amount, currency),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return payment
}

func newAuthorizedPayment(
	t *testing.T,
	amount int64,
	currency Currency,
) *Payment {
	t.Helper()

	payment := newPendingPayment(t, amount, currency)

	if err := payment.Authorize(); err != nil {
		t.Fatalf("Authorize() error = %v", err)
	}

	return payment
}

func mustMoney(t *testing.T, amount int64, currency Currency) Money {
	t.Helper()

	money, err := NewMoney(amount, currency)
	if err != nil {
		t.Fatalf("NewMoney() error = %v", err)
	}

	return money
}

func mustPaymentState(
	t *testing.T,
	status Status,
) PaymentState {
	t.Helper()

	amount := mustMoney(t, 10000, "EUR")

	switch status {
	case StatusPending:
		return PaymentState{
			ID:      "payment-test",
			Amount:  amount,
			Status:  StatusPending,
			Version: 1,
		}

	case StatusAuthorized:
		return PaymentState{
			ID:               "payment-test",
			Amount:           amount,
			Status:           StatusAuthorized,
			AuthorizedAmount: 10000,
			Version:          2,
		}

	case StatusPartiallyCaptured:
		return PaymentState{
			ID:               "payment-test",
			Amount:           amount,
			Status:           StatusPartiallyCaptured,
			AuthorizedAmount: 10000,
			CapturedAmount:   4000,
			Version:          3,
		}

	case StatusCaptured:
		return PaymentState{
			ID:               "payment-test",
			Amount:           amount,
			Status:           StatusCaptured,
			AuthorizedAmount: 10000,
			CapturedAmount:   10000,
			Version:          4,
		}

	case StatusPartiallyRefunded:
		return PaymentState{
			ID:               "payment-test",
			Amount:           amount,
			Status:           StatusPartiallyRefunded,
			AuthorizedAmount: 10000,
			CapturedAmount:   10000,
			RefundedAmount:   3000,
			Version:          5,
		}

	case StatusRefunded:
		return PaymentState{
			ID:               "payment-test",
			Amount:           amount,
			Status:           StatusRefunded,
			AuthorizedAmount: 10000,
			CapturedAmount:   10000,
			RefundedAmount:   10000,
			Version:          6,
		}

	case StatusFailed:
		return PaymentState{
			ID:      "payment-test",
			Amount:  amount,
			Status:  StatusFailed,
			Version: 2,
		}

	case StatusCancelled:
		return PaymentState{
			ID:               "payment-test",
			Amount:           amount,
			Status:           StatusCancelled,
			AuthorizedAmount: 10000,
			Version:          3,
		}
	}

	t.Fatalf("unsupported status %q", status)
	return PaymentState{}
}

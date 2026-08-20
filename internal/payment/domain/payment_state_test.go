package domain

import "testing"

func TestPaymentState_PartiallyRefundedIsValid(t *testing.T) {
	state := mustPaymentState(t, StatusPartiallyRefunded)

	if err := state.Validate(); err != nil {
		t.Fatalf(
			"PaymentState.Validate() error = %v, want nil",
			err,
		)
	}
}

func TestPaymentState_ValidLifecycleStates(t *testing.T) {
	statuses := []Status{
		StatusPending,
		StatusAuthorized,
		StatusPartiallyCaptured,
		StatusCaptured,
		StatusPartiallyRefunded,
		StatusRefunded,
		StatusFailed,
		StatusCancelled,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			state := mustPaymentState(t, status)

			if err := state.Validate(); err != nil {
				t.Fatalf(
					"PaymentState.Validate() error = %v, want nil",
					err,
				)
			}
		})
	}
}

package application

import (
	"reflect"
	"testing"
	"time"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
)

func TestCompare_EquivalentResultsIgnorePaymentSliceOrder(t *testing.T) {
	first := comparisonResult(
		time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC),
		paymentState("payment-1", paymentdomain.StatusCaptured, 10000, 10000, 10000, 0, 3),
		paymentState("payment-2", paymentdomain.StatusPending, 5000, 0, 0, 0, 1),
	)
	second := comparisonResult(
		time.Date(2026, 8, 24, 13, 0, 0, 0, time.FixedZone("CET", 3600)),
		first.Payments[1],
		first.Payments[0],
	)

	got := Compare(first, second)
	if !got.Equivalent {
		t.Fatalf("Compare() = %+v, want equivalent", got)
	}
	if len(got.Differences) != 0 {
		t.Fatalf("Differences = %+v, want none", got.Differences)
	}
}

func TestCompare_ReportsStableObservableDifferences(t *testing.T) {
	expected := comparisonResult(
		time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC),
		paymentState("payment-1", paymentdomain.StatusCaptured, 10000, 10000, 10000, 0, 3),
		paymentState("payment-2", paymentdomain.StatusPending, 5000, 0, 0, 0, 1),
	)
	actual := comparisonResult(
		time.Date(2026, 8, 24, 12, 1, 0, 0, time.UTC),
		paymentState("payment-1", paymentdomain.StatusPartiallyCaptured, 10000, 10000, 4000, 0, 2),
		paymentState("payment-3", paymentdomain.StatusPending, 2500, 0, 0, 0, 1),
	)

	got := Compare(expected, actual)
	want := Comparison{
		Equivalent: false,
		Differences: []Difference{
			{Path: "current_virtual_time", Expected: "2026-08-24T12:00:00Z", Actual: "2026-08-24T12:01:00Z"},
			{Path: `payments["payment-1"].captured_amount`, Expected: "10000", Actual: "4000"},
			{Path: `payments["payment-1"].status`, Expected: "captured", Actual: "partially_captured"},
			{Path: `payments["payment-1"].version`, Expected: "3", Actual: "2"},
			{Path: `payments["payment-2"]`, Expected: "present", Actual: missingValue},
			{Path: `payments["payment-3"]`, Expected: unexpectedValue, Actual: "present"},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Compare() = %+v, want %+v", got, want)
	}
}

func TestCompare_IsDeterministic(t *testing.T) {
	expected := comparisonResult(time.Time{}, paymentState("payment-1", paymentdomain.StatusPending, 100, 0, 0, 0, 1))
	actual := comparisonResult(time.Now(), paymentState("payment-1", paymentdomain.StatusCaptured, 100, 100, 100, 0, 3))

	first := Compare(expected, actual)
	second := Compare(expected, actual)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated Compare() calls differ: first = %+v, second = %+v", first, second)
	}
}

func comparisonResult(at time.Time, payments ...paymentdomain.PaymentState) Result {
	return Result{CurrentVirtualTime: at, Payments: payments}
}

func paymentState(
	id paymentdomain.ID,
	status paymentdomain.Status,
	amount, authorized, captured, refunded int64,
	version uint64,
) paymentdomain.PaymentState {
	money, err := paymentdomain.NewMoney(amount, "EUR")
	if err != nil {
		panic(err)
	}

	return paymentdomain.PaymentState{
		ID:               id,
		Amount:           money,
		Status:           status,
		AuthorizedAmount: authorized,
		CapturedAmount:   captured,
		RefundedAmount:   refunded,
		Version:          version,
	}
}

package application

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
)

const (
	missingValue    = "<missing>"
	unexpectedValue = "<unexpected>"
)

// Difference identifies one observable behaviour mismatch between two
// execution results.
type Difference struct {
	Path     string
	Expected string
	Actual   string
}

// Comparison reports whether two execution results are behaviourally
// equivalent and, when they are not, why they differ.
type Comparison struct {
	Equivalent  bool
	Differences []Difference
}

// Compare compares observable execution outcomes without considering
// implementation details or the order of payment result slices.
func Compare(expected, actual Result) Comparison {
	differences := make([]Difference, 0)

	if !expected.CurrentVirtualTime.Equal(actual.CurrentVirtualTime) {
		differences = append(differences, Difference{
			Path:     "current_virtual_time",
			Expected: formatTime(expected.CurrentVirtualTime),
			Actual:   formatTime(actual.CurrentVirtualTime),
		})
	}

	expectedPayments := indexPayments(expected.Payments)
	actualPayments := indexPayments(actual.Payments)

	for id, expectedPayment := range expectedPayments {
		actualPayment, exists := actualPayments[id]
		if !exists {
			differences = append(differences, Difference{
				Path:     paymentPath(id),
				Expected: "present",
				Actual:   missingValue,
			})
			continue
		}

		comparePayment(&differences, id, expectedPayment, actualPayment)
	}

	for id := range actualPayments {
		if _, exists := expectedPayments[id]; exists {
			continue
		}

		differences = append(differences, Difference{
			Path:     paymentPath(id),
			Expected: unexpectedValue,
			Actual:   "present",
		})
	}

	sort.Slice(differences, func(i, j int) bool {
		if differences[i].Path != differences[j].Path {
			return differences[i].Path < differences[j].Path
		}
		if differences[i].Expected != differences[j].Expected {
			return differences[i].Expected < differences[j].Expected
		}
		return differences[i].Actual < differences[j].Actual
	})

	return Comparison{
		Equivalent:  len(differences) == 0,
		Differences: differences,
	}
}

func indexPayments(payments []paymentdomain.PaymentState) map[paymentdomain.ID]paymentdomain.PaymentState {
	indexed := make(map[paymentdomain.ID]paymentdomain.PaymentState, len(payments))
	for _, payment := range payments {
		indexed[payment.ID] = payment
	}

	return indexed
}

func comparePayment(
	differences *[]Difference,
	id paymentdomain.ID,
	expected, actual paymentdomain.PaymentState,
) {
	path := paymentPath(id)
	compareValue(differences, path+".status", string(expected.Status), string(actual.Status))
	compareValue(
		differences,
		path+".amount",
		formatInt(expected.Amount.Amount()),
		formatInt(actual.Amount.Amount()),
	)
	compareValue(
		differences,
		path+".currency",
		string(expected.Amount.Currency()),
		string(actual.Amount.Currency()),
	)
	compareValue(
		differences,
		path+".authorized_amount",
		formatInt(expected.AuthorizedAmount),
		formatInt(actual.AuthorizedAmount),
	)
	compareValue(
		differences,
		path+".captured_amount",
		formatInt(expected.CapturedAmount),
		formatInt(actual.CapturedAmount),
	)
	compareValue(
		differences,
		path+".refunded_amount",
		formatInt(expected.RefundedAmount),
		formatInt(actual.RefundedAmount),
	)
	compareValue(
		differences,
		path+".version",
		formatUint(expected.Version),
		formatUint(actual.Version),
	)
}

func compareValue(differences *[]Difference, path, expected, actual string) {
	if expected == actual {
		return
	}

	*differences = append(*differences, Difference{
		Path:     path,
		Expected: expected,
		Actual:   actual,
	})
}

func paymentPath(id paymentdomain.ID) string {
	return fmt.Sprintf("payments[%q]", id)
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func formatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}

func formatUint(value uint64) string {
	return strconv.FormatUint(value, 10)
}

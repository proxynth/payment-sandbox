package application

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/payment/domain"
)

// Invariant: a complete persisted event history reproduces the exact current
// monetary state without consulting the current-state repository.
func TestReconstructFromEventsRestoresCurrentPaymentState(t *testing.T) {
	money, err := domain.NewMoney(1000, "EUR")
	if err != nil {
		t.Fatal(err)
	}
	payment, err := domain.New("payment-1", money)
	if err != nil {
		t.Fatal(err)
	}
	events := []domain.BusinessEvent{reconstructionEvent(t, payment, domain.EventPaymentCreated)}
	if err := payment.Authorize(); err != nil {
		t.Fatal(err)
	}
	events = append(events, reconstructionEvent(t, payment, domain.EventPaymentAuthorized))
	capture, _ := domain.NewMoney(400, "EUR")
	if err := payment.Capture(capture); err != nil {
		t.Fatal(err)
	}
	events = append(events, reconstructionEvent(t, payment, domain.EventPaymentCaptured))

	reconstructed, err := ReconstructFromEvents(events)
	if err != nil {
		t.Fatal(err)
	}
	if reconstructed.ID() != payment.ID() || reconstructed.Status() != payment.Status() || reconstructed.Version() != payment.Version() || reconstructed.CapturedAmount() != payment.CapturedAmount() {
		t.Fatalf("reconstructed payment = %#v, want state of %#v", reconstructed, payment)
	}
}

func TestReconstructFromEventsRejectsMissingOrLegacySnapshots(t *testing.T) {
	if _, err := ReconstructFromEvents(nil); !errors.Is(err, ErrEmptyEventHistory) {
		t.Fatalf("empty history error = %v", err)
	}
	event, err := domain.NewBusinessEvent("event-1", "payment-1", domain.EventPaymentCreated, time.Unix(1, 0), 1, "correlation", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReconstructFromEvents([]domain.BusinessEvent{event}); !errors.Is(err, ErrInconsistentEventHistory) {
		t.Fatalf("legacy history error = %v", err)
	}
}

func TestReconstructFromEventsRejectsVersionGap(t *testing.T) {
	money, _ := domain.NewMoney(1000, "EUR")
	payment, _ := domain.New("payment-1", money)
	event := reconstructionEvent(t, payment, domain.EventPaymentCreated)
	// Make the event look like the second version: its payload is still a
	// valid state, but the history cannot prove that version one was retained.
	payload := event.Payload()
	gapped, err := domain.NewBusinessEventWithPayload("event-2", "payment-1", domain.EventPaymentAuthorized, time.Unix(2, 0), 2, "correlation", "", payload)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReconstructFromEvents([]domain.BusinessEvent{gapped}); !errors.Is(err, ErrIncompleteEventHistory) {
		t.Fatalf("gapped history error = %v", err)
	}
}

func reconstructionEvent(t *testing.T, payment *domain.Payment, eventType domain.EventType) domain.BusinessEvent {
	t.Helper()
	payload, err := json.Marshal(domain.NewEventSnapshot(payment))
	if err != nil {
		t.Fatal(err)
	}
	event, err := domain.NewBusinessEventWithPayload(
		domain.EventID("event-"+string(rune('0'+payment.Version()))), payment.ID(), eventType,
		time.Unix(int64(payment.Version()), 0), payment.Version(), "correlation", "", payload,
	)
	if err != nil {
		t.Fatal(err)
	}
	return event
}

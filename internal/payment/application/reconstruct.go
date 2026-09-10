package application

import (
	"encoding/json"
	"errors"
	"fmt"

	"proxynth/payment-sandbox/internal/payment/domain"
)

var (
	ErrEmptyEventHistory        = errors.New("payment event history is empty")
	ErrIncompleteEventHistory   = errors.New("payment event history is incomplete")
	ErrInconsistentEventHistory = errors.New("payment event history is inconsistent")
)

// ReconstructFromEvents recreates the current payment state from the complete
// ordered event history. It deliberately rejects legacy events without a
// snapshot and histories whose aggregate versions are not contiguous.
func ReconstructFromEvents(events []domain.BusinessEvent) (*domain.Payment, error) {
	if len(events) == 0 {
		return nil, ErrEmptyEventHistory
	}
	var aggregateID domain.ID
	var payment *domain.Payment
	for index, event := range events {
		if err := event.Validate(); err != nil {
			return nil, fmt.Errorf("%w: invalid event %q: %w", ErrInconsistentEventHistory, event.ID(), err)
		}
		if index == 0 {
			aggregateID = event.AggregateID()
		}
		if event.AggregateID() != aggregateID || event.AggregateVersion() != uint64(index+1) {
			return nil, fmt.Errorf("%w: event %q has aggregate version %d at position %d", ErrIncompleteEventHistory, event.ID(), event.AggregateVersion(), index+1)
		}
		var snapshot domain.EventSnapshot
		if err := json.Unmarshal(event.Payload(), &snapshot); err != nil {
			return nil, fmt.Errorf("%w: event %q snapshot: %w", ErrInconsistentEventHistory, event.ID(), err)
		}
		if snapshot.PaymentID != aggregateID || snapshot.Version != event.AggregateVersion() {
			return nil, fmt.Errorf("%w: event %q snapshot identity", ErrInconsistentEventHistory, event.ID())
		}
		restored, err := snapshot.Restore()
		if err != nil {
			return nil, fmt.Errorf("%w: event %q snapshot: %w", ErrInconsistentEventHistory, event.ID(), err)
		}
		payment = restored
	}
	return payment, nil
}

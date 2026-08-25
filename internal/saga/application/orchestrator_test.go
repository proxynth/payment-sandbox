package application

import (
	"context"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/saga/domain"
)

func TestOrchestratorPublishesStepsAndCompensatesAfterFailure(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	publisher := &memoryPublisher{}
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	orchestrator, err := NewOrchestrator(store, publisher, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}

	if err := orchestrator.Start(ctx, "saga-payment-42", "payment-42", 42); err != nil {
		t.Fatal(err)
	}
	if len(publisher.messages) != 1 || publisher.messages[0].Step != domain.StepAuthorize {
		t.Fatalf("messages = %+v", publisher.messages)
	}

	if err := orchestrator.Handle(ctx, publisher.messages[0], executorFunc(func(context.Context, domain.Message) (Outcome, error) { return OutcomeSucceeded, nil })); err != nil {
		t.Fatal(err)
	}
	if publisher.messages[1].Step != domain.StepCapture {
		t.Fatalf("next step = %s", publisher.messages[1].Step)
	}

	if err := orchestrator.Handle(ctx, publisher.messages[1], executorFunc(func(context.Context, domain.Message) (Outcome, error) { return OutcomeFailed, nil })); err != nil {
		t.Fatal(err)
	}
	if publisher.messages[2].Step != domain.StepCancel {
		t.Fatalf("compensation = %s", publisher.messages[2].Step)
	}
	if err := orchestrator.Handle(ctx, publisher.messages[2], executorFunc(func(context.Context, domain.Message) (Outcome, error) { return OutcomeSucceeded, nil })); err != nil {
		t.Fatal(err)
	}

	instance, err := store.Find(ctx, "saga-payment-42")
	if err != nil {
		t.Fatal(err)
	}
	if instance.Status != domain.StatusFailed {
		t.Fatalf("status = %s", instance.Status)
	}
}

func TestOrchestratorIgnoresDuplicateDelivery(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	publisher := &memoryPublisher{}
	orchestrator, err := NewOrchestrator(store, publisher, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if err := orchestrator.Start(ctx, "saga-1", "payment-1", 7); err != nil {
		t.Fatal(err)
	}
	message := publisher.messages[0]
	calls := 0
	executor := executorFunc(func(context.Context, domain.Message) (Outcome, error) { calls++; return OutcomeSucceeded, nil })
	if err := orchestrator.Handle(ctx, message, executor); err != nil {
		t.Fatal(err)
	}
	if err := orchestrator.Handle(ctx, message, executor); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("executor calls = %d, want 1", calls)
	}
}

type executorFunc func(context.Context, domain.Message) (Outcome, error)

func (f executorFunc) Execute(ctx context.Context, m domain.Message) (Outcome, error) {
	return f(ctx, m)
}

type memoryPublisher struct{ messages []domain.Message }

func (p *memoryPublisher) Publish(_ context.Context, message domain.Message) error {
	p.messages = append(p.messages, message)
	return nil
}

type memoryStore struct{ instances map[domain.ID]domain.Instance }

func newMemoryStore() *memoryStore {
	return &memoryStore{instances: make(map[domain.ID]domain.Instance)}
}
func (s *memoryStore) Save(_ context.Context, instance domain.Instance) error {
	s.instances[instance.ID] = instance
	return nil
}
func (s *memoryStore) Find(_ context.Context, id domain.ID) (domain.Instance, error) {
	return s.instances[id], nil
}

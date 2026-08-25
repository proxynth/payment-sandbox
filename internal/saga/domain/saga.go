package domain

import (
	"errors"
	"time"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
)

type ID string

type Step string

const (
	StepAuthorize Step = "authorize"
	StepCapture   Step = "capture"
	StepRefund    Step = "refund"
	StepCancel    Step = "cancel"
)

type Status string

const (
	StatusRunning      Status = "running"
	StatusCompensating Status = "compensating"
	StatusCompleted    Status = "completed"
	StatusFailed       Status = "failed"
)

var (
	ErrInvalidID         = errors.New("invalid saga id")
	ErrInvalidStep       = errors.New("invalid saga step")
	ErrInvalidStatus     = errors.New("invalid saga status")
	ErrInvalidMessage    = errors.New("invalid saga message")
	ErrInvalidTransition = errors.New("invalid saga transition")
)

type Message struct {
	ID          string
	SagaID      ID
	PaymentID   paymentdomain.ID
	Step        Step
	Payload     []byte
	Seed        uint64
	ScheduledAt time.Time
	VirtualAt   time.Time
	Attempt     uint64
}

func (m Message) Validate() error {
	if m.ID == "" || m.SagaID == "" || m.PaymentID == "" || !m.Step.Valid() || m.ScheduledAt.IsZero() || m.VirtualAt.IsZero() || m.Attempt == 0 {
		return ErrInvalidMessage
	}
	return nil
}

func (s Step) Valid() bool {
	switch s {
	case StepAuthorize, StepCapture, StepRefund, StepCancel:
		return true
	default:
		return false
	}
}

func (s Status) Valid() bool {
	switch s {
	case StatusRunning, StatusCompensating, StatusCompleted, StatusFailed:
		return true
	default:
		return false
	}
}

type Instance struct {
	ID             ID
	PaymentID      paymentdomain.ID
	Status         Status
	CurrentStep    Step
	CompletedSteps []Step
	Compensation   []Step
	Seed           uint64
	Version        uint64
	UpdatedAt      time.Time
}

func New(id ID, paymentID paymentdomain.ID, seed uint64, at time.Time) (Instance, error) {
	if id == "" {
		return Instance{}, ErrInvalidID
	}
	if paymentID == "" || at.IsZero() {
		return Instance{}, ErrInvalidTransition
	}
	return Instance{ID: id, PaymentID: paymentID, Status: StatusRunning, CurrentStep: StepAuthorize, Seed: seed, Version: 1, UpdatedAt: at.UTC()}, nil
}

func (i Instance) Validate() error {
	if i.ID == "" || i.PaymentID == "" || !i.Status.Valid() || !i.CurrentStep.Valid() || i.Version == 0 || i.UpdatedAt.IsZero() {
		return ErrInvalidTransition
	}
	return nil
}

func (i *Instance) ApplySuccess(step Step, at time.Time) error {
	if !step.Valid() || i.Status != StatusRunning || step != i.CurrentStep {
		return ErrInvalidTransition
	}
	if i.contains(step) {
		return nil
	}
	i.CompletedSteps = append(i.CompletedSteps, step)
	i.Version++
	i.UpdatedAt = at.UTC()
	switch step {
	case StepAuthorize:
		i.CurrentStep = StepCapture
	case StepCapture:
		i.Status = StatusCompleted
	}
	return nil
}

func (i *Instance) BeginCompensation(step Step, at time.Time) error {
	if i.Status != StatusRunning || !step.Valid() {
		return ErrInvalidTransition
	}
	i.Status = StatusCompensating
	i.Compensation = compensationFor(i.CompletedSteps)
	if len(i.Compensation) == 0 {
		i.Status = StatusFailed
		i.Version++
		i.UpdatedAt = at.UTC()
		return nil
	}
	i.CurrentStep = i.Compensation[0]
	i.Version++
	i.UpdatedAt = at.UTC()
	return nil
}

func (i *Instance) ApplyCompensationSuccess(step Step, at time.Time) error {
	if i.Status != StatusCompensating || len(i.Compensation) == 0 || step != i.CurrentStep {
		return ErrInvalidTransition
	}
	i.Compensation = i.Compensation[1:]
	i.Version++
	i.UpdatedAt = at.UTC()
	if len(i.Compensation) == 0 {
		i.Status = StatusFailed
		return nil
	}
	i.CurrentStep = i.Compensation[0]
	return nil
}

func (i Instance) contains(step Step) bool {
	for _, completed := range i.CompletedSteps {
		if completed == step {
			return true
		}
	}
	return false
}

func compensationFor(completed []Step) []Step {
	for _, step := range completed {
		if step == StepCapture {
			return []Step{StepRefund, StepCancel}
		}
	}
	for _, step := range completed {
		if step == StepAuthorize {
			return []Step{StepCancel}
		}
	}
	return nil
}

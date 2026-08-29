package application

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	paymentapplication "proxynth/payment-sandbox/internal/payment/application"
	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	paymentworkflowapplication "proxynth/payment-sandbox/internal/paymentworkflow/application"
	paymentworkflowdomain "proxynth/payment-sandbox/internal/paymentworkflow/domain"
	"proxynth/payment-sandbox/internal/platform/clock"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
	replaydomain "proxynth/payment-sandbox/internal/replay/domain"
	schedulerapplication "proxynth/payment-sandbox/internal/scheduler/application"
	schedulerdomain "proxynth/payment-sandbox/internal/scheduler/domain"
)

const providerAsyncJobType schedulerdomain.JobType = "provider.async"
const workflowStepJobType schedulerdomain.JobType = "saga.step"

// Result contains the deterministic state produced by one scenario execution.
type Result struct {
	ScenarioID                 replaydomain.ScenarioID
	Provider                   replaydomain.ProviderConfiguration
	DeterministicConfiguration replaydomain.DeterministicConfiguration
	CurrentVirtualTime         time.Time
	Payments                   []paymentdomain.PaymentState
	AsyncOperations            []providerdomain.AsyncOperation
}

// ProviderRegistry resolves the provider selected by a scenario.
type ProviderRegistry interface {
	Resolve(providerdomain.ProviderID) (providerdomain.Provider, error)
}

// Runner executes one scenario against the production payment application
// services and an execution-scoped in-memory repository.
type Runner struct {
	providers ProviderRegistry
}

func NewRunner(providers ProviderRegistry) *Runner {
	return &Runner{providers: providers}
}

func (r *Runner) Run(ctx context.Context, scenario replaydomain.Scenario) (Result, error) {
	if err := scenario.Validate(); err != nil {
		return Result{}, err
	}

	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if r.providers == nil {
		return Result{}, ErrNilProviderRegistry
	}

	provider, err := r.providers.Resolve(scenario.Provider.ID)
	if err != nil {
		return Result{}, err
	}
	if configurable, ok := provider.(providerdomain.ConfigurableProvider); ok {
		provider, err = configurable.Configure(scenario.Provider.Profile, scenario.DeterministicConfiguration.Seed)
		if err != nil {
			return Result{}, err
		}
	}

	virtualClock, err := clock.NewVirtualClock(scenario.InitialVirtualTime)
	if err != nil {
		return Result{}, err
	}

	repository := newMemoryRepository()
	for _, state := range scenario.InitialPayments {
		payment, err := paymentdomain.Restore(state)
		if err != nil {
			return Result{}, err
		}

		repository.payments[payment.ID()] = payment
	}

	jobs := newScenarioJobRepository()
	workflowStore := newScenarioSagaStore()
	workflowPublisher := &scenarioSagaPublisher{jobs: jobs}
	workflowOrchestrator, err := paymentworkflowapplication.NewOrchestrator(workflowStore, workflowPublisher, virtualClock.Now)
	if err != nil {
		return Result{}, err
	}
	workflowExecutor, err := paymentworkflowapplication.NewPaymentExecutor(repository, provider, virtualClock)
	if err != nil {
		return Result{}, err
	}
	services := &commandServices{repository: repository, provider: provider, clock: virtualClock, jobs: jobs, workflow: workflowOrchestrator, workflowExecutor: workflowExecutor}
	worker, err := schedulerapplication.NewWorker(jobs, map[schedulerdomain.JobType]schedulerapplication.JobHandler{providerAsyncJobType: services.handleAsync, workflowStepJobType: services.handleSaga})
	if err != nil {
		return Result{}, err
	}
	scheduler, err := schedulerapplication.NewScheduler(jobs, workerDispatcher{worker}, virtualClock, virtualClock, schedulerapplication.Config{Owner: "replay", BatchSize: 100, LeaseDuration: time.Minute})
	if err != nil {
		return Result{}, err
	}
	services.scheduler = scheduler
	asyncOperations := make([]providerdomain.AsyncOperation, 0)
	for index, command := range scenario.Commands {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}

		operations, err := services.execute(ctx, command)
		if err != nil {
			return Result{}, fmt.Errorf(
				"execute scenario command %d (%s): %w",
				index,
				command.Type,
				err,
			)
		}
		asyncOperations = appendAsyncOperations(asyncOperations, operations...)
	}

	return Result{
		ScenarioID:                 scenario.ID,
		Provider:                   scenario.Provider,
		DeterministicConfiguration: scenario.DeterministicConfiguration,
		CurrentVirtualTime:         virtualClock.Now(),
		Payments:                   repository.states(),
		AsyncOperations:            asyncOperations,
	}, nil
}

type commandServices struct {
	repository       *memoryRepository
	provider         providerdomain.Provider
	clock            *clock.VirtualClock
	jobs             *scenarioJobRepository
	scheduler        *schedulerapplication.Scheduler
	workflow         *paymentworkflowapplication.Orchestrator
	workflowExecutor *paymentworkflowapplication.PaymentExecutor
}

func (s *commandServices) handleSaga(ctx context.Context, payload []byte) error {
	var message paymentworkflowdomain.Message
	if err := json.Unmarshal(payload, &message); err != nil {
		return err
	}
	return s.workflow.Handle(ctx, message, s.workflowExecutor)
}

func (s *commandServices) execute(
	ctx context.Context,
	command replaydomain.Command,
) ([]providerdomain.AsyncOperation, error) {
	switch command.Type {
	case replaydomain.CommandCreatePayment:
		_, err := paymentapplication.NewCreatePayment(s.repository).Execute(
			ctx,
			paymentapplication.CreatePaymentCommand{
				ID:       command.PaymentID,
				Amount:   command.Amount.Amount(),
				Currency: command.Amount.Currency(),
			},
		)
		return nil, err
	case replaydomain.CommandStartSaga:
		payload, err := json.Marshal(struct {
			Amount   int64                  `json:"amount"`
			Currency paymentdomain.Currency `json:"currency"`
		}{command.Amount.Amount(), command.Amount.Currency()})
		if err != nil {
			return nil, err
		}
		if err := s.workflow.StartWithPayload(ctx, paymentworkflowdomain.ID("saga:"+string(command.PaymentID)), string(command.PaymentID), payload, 0); err != nil {
			return nil, err
		}
		if err := s.scheduler.Tick(ctx); err != nil {
			return nil, err
		}
		if err := s.scheduler.Tick(ctx); err != nil {
			return nil, err
		}
		return nil, nil
	case replaydomain.CommandAuthorize:
		payment, err := s.repository.FindByID(ctx, command.PaymentID)
		if err != nil {
			return nil, err
		}
		result, err := s.authorize(ctx, payment)
		if err != nil {
			return nil, err
		}
		return s.apply(ctx, command, payment, result, func() error {
			_, err := paymentapplication.NewAuthorizePayment(s.repository).Execute(ctx, paymentapplication.AuthorizePaymentCommand{PaymentID: command.PaymentID})
			return err
		})
	case replaydomain.CommandCapture:
		payment, err := s.repository.FindByID(ctx, command.PaymentID)
		if err != nil {
			return nil, err
		}
		result, err := s.capture(ctx, payment, command.Amount)
		if err != nil {
			return nil, err
		}
		return s.apply(ctx, command, payment, result, func() error {
			_, err := paymentapplication.NewCapturePayment(s.repository).Execute(ctx, paymentapplication.CapturePaymentCommand{
				PaymentID: command.PaymentID,
				Amount:    command.Amount.Amount(),
				Currency:  command.Amount.Currency(),
			})
			return err
		})
	case replaydomain.CommandRefund:
		payment, err := s.repository.FindByID(ctx, command.PaymentID)
		if err != nil {
			return nil, err
		}
		result, err := s.refund(ctx, payment, command.Amount)
		if err != nil {
			return nil, err
		}
		return s.apply(ctx, command, payment, result, func() error {
			_, err := paymentapplication.NewRefundPayment(s.repository).Execute(ctx, paymentapplication.RefundPaymentCommand{
				PaymentID: command.PaymentID,
				Amount:    command.Amount.Amount(),
				Currency:  command.Amount.Currency(),
			})
			return err
		})
	case replaydomain.CommandCancel:
		payment, err := s.repository.FindByID(ctx, command.PaymentID)
		if err != nil {
			return nil, err
		}
		result, err := s.cancel(ctx, payment)
		if err != nil {
			return nil, err
		}
		return s.apply(ctx, command, payment, result, func() error {
			_, err := paymentapplication.NewCancelPayment(s.repository).Execute(ctx, paymentapplication.CancelPaymentCommand{PaymentID: command.PaymentID})
			return err
		})
	case replaydomain.CommandAdvanceTime:
		return nil, s.clock.Advance(command.Duration)
	case replaydomain.CommandExecuteAsync:
		job, exists := s.jobs.jobs[schedulerdomain.JobID(command.OperationID)]
		if !exists {
			return nil, ErrAsyncOperationNotFound
		}
		if job.Status() == schedulerdomain.JobPending && job.NextAttemptAt().After(s.clock.Now()) {
			return nil, ErrAsyncOperationNotDue
		}
		if err := s.scheduler.Tick(ctx); err != nil {
			return nil, err
		}
		if job.Status() != schedulerdomain.JobCompleted {
			return nil, ErrAsyncOperationNotDue
		}
		return nil, nil
	default:
		return nil, replaydomain.ErrInvalidCommand
	}
}

func (s *commandServices) apply(ctx context.Context, command replaydomain.Command, payment *paymentdomain.Payment, result providerdomain.OperationResult, transition func() error) ([]providerdomain.AsyncOperation, error) {
	switch result.Outcome {
	case providerdomain.OutcomeSucceeded:
		if err := transition(); err != nil {
			return nil, err
		}
	case providerdomain.OutcomeFailed:
		if command.Type == replaydomain.CommandAuthorize {
			if _, err := paymentapplication.NewFailPayment(s.repository).Execute(ctx, paymentapplication.FailPaymentCommand{PaymentID: command.PaymentID}); err != nil {
				return nil, err
			}
		} else {
			return nil, ErrProviderOperationFailed
		}
	case providerdomain.OutcomePending:
		for _, operation := range result.AsyncOperations {
			payload, err := json.Marshal(operation)
			if err != nil {
				return nil, err
			}
			job, err := schedulerdomain.NewJob(schedulerdomain.JobID(operation.ID), providerAsyncJobType, payload, operation.ScheduledAt)
			if err != nil {
				return nil, err
			}
			if err := s.jobs.Save(ctx, &job); err != nil {
				return nil, err
			}
		}
	}
	return appendAsyncOperations(nil, result.AsyncOperations...), nil
}

func (s *commandServices) handleAsync(ctx context.Context, payload []byte) error {
	var operation providerdomain.AsyncOperation
	if err := json.Unmarshal(payload, &operation); err != nil {
		return err
	}
	asyncProvider, ok := s.provider.(providerdomain.AsyncExecutor)
	if !ok {
		return ErrAsyncOperationNotFound
	}
	payment, err := s.repository.FindByID(ctx, operation.PaymentID)
	if err != nil {
		return err
	}
	result, err := validateProviderResult(asyncProvider.ExecuteAsync(ctx, operation))
	if err != nil {
		return err
	}
	command := replaydomain.Command{Type: replaydomain.CommandType(operation.Type), PaymentID: operation.PaymentID}
	_, err = s.apply(ctx, command, payment, result, func() error { return s.transition(ctx, command, payment) })
	return err
}

func (s commandServices) transition(ctx context.Context, command replaydomain.Command, payment *paymentdomain.Payment) error {
	switch command.Type {
	case replaydomain.CommandAuthorize:
		_, err := paymentapplication.NewAuthorizePayment(s.repository).Execute(ctx, paymentapplication.AuthorizePaymentCommand{PaymentID: payment.ID()})
		return err
	case replaydomain.CommandCapture:
		_, err := paymentapplication.NewCapturePayment(s.repository).Execute(ctx, paymentapplication.CapturePaymentCommand{PaymentID: payment.ID(), Amount: command.Amount.Amount(), Currency: command.Amount.Currency()})
		return err
	case replaydomain.CommandRefund:
		_, err := paymentapplication.NewRefundPayment(s.repository).Execute(ctx, paymentapplication.RefundPaymentCommand{PaymentID: payment.ID(), Amount: command.Amount.Amount(), Currency: command.Amount.Currency()})
		return err
	case replaydomain.CommandCancel:
		_, err := paymentapplication.NewCancelPayment(s.repository).Execute(ctx, paymentapplication.CancelPaymentCommand{PaymentID: payment.ID()})
		return err
	default:
		return replaydomain.ErrInvalidCommand
	}
}

func (s commandServices) authorize(ctx context.Context, payment *paymentdomain.Payment) (providerResult providerdomain.OperationResult, err error) {
	return validateProviderResult(s.provider.Authorize(ctx, providerdomain.AuthorizeRequest{
		Payment: paymentSnapshot(payment),
		At:      s.clock.Now(),
	}))
}

func (s commandServices) capture(ctx context.Context, payment *paymentdomain.Payment, amount paymentdomain.Money) (providerResult providerdomain.OperationResult, err error) {
	return validateProviderResult(s.provider.Capture(ctx, providerdomain.CaptureRequest{
		Payment: paymentSnapshot(payment),
		Amount:  amount,
		At:      s.clock.Now(),
	}))
}

func (s commandServices) refund(ctx context.Context, payment *paymentdomain.Payment, amount paymentdomain.Money) (providerResult providerdomain.OperationResult, err error) {
	return validateProviderResult(s.provider.Refund(ctx, providerdomain.RefundRequest{
		Payment: paymentSnapshot(payment),
		Amount:  amount,
		At:      s.clock.Now(),
	}))
}

func (s commandServices) cancel(ctx context.Context, payment *paymentdomain.Payment) (providerResult providerdomain.OperationResult, err error) {
	return validateProviderResult(s.provider.Cancel(ctx, providerdomain.CancelRequest{
		Payment: paymentSnapshot(payment),
		At:      s.clock.Now(),
	}))
}

func paymentSnapshot(payment *paymentdomain.Payment) providerdomain.PaymentSnapshot {
	return providerdomain.PaymentSnapshot{
		ID:      payment.ID(),
		Amount:  payment.Amount(),
		Status:  payment.Status(),
		Version: payment.Version(),
	}
}

func validateProviderResult(result providerdomain.OperationResult, err error) (providerdomain.OperationResult, error) {
	if err != nil {
		return providerdomain.OperationResult{}, err
	}

	if err := result.Validate(); err != nil {
		return providerdomain.OperationResult{}, err
	}

	return result, nil
}

func appendAsyncOperations(
	operations []providerdomain.AsyncOperation,
	additional ...providerdomain.AsyncOperation,
) []providerdomain.AsyncOperation {
	for _, operation := range additional {
		operation.Payload = append([]byte(nil), operation.Payload...)
		operations = append(operations, operation)
	}

	return operations
}

type workerDispatcher struct{ worker *schedulerapplication.Worker }

func (d workerDispatcher) Dispatch(ctx context.Context, job *schedulerdomain.Job) error {
	return d.worker.Execute(ctx, job)
}

// scenarioJobRepository gives replay the same durable job lifecycle as the
// runtime scheduler while keeping one replay isolated from another. The
// production scheduler persists the equivalent records in its repository.
type scenarioJobRepository struct {
	jobs map[schedulerdomain.JobID]*schedulerdomain.Job
}

func newScenarioJobRepository() *scenarioJobRepository {
	return &scenarioJobRepository{jobs: make(map[schedulerdomain.JobID]*schedulerdomain.Job)}
}

func (r *scenarioJobRepository) Save(ctx context.Context, job *schedulerdomain.Job) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if job == nil {
		return schedulerapplication.ErrNilAcquiredJob
	}
	r.jobs[job.ID()] = job
	return nil
}

func (r *scenarioJobRepository) FindExecutable(ctx context.Context, at time.Time, limit int) ([]*schedulerdomain.Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	jobs := make([]*schedulerdomain.Job, 0, limit)
	for _, job := range r.jobs {
		if len(jobs) == limit {
			break
		}
		if job.Status() == schedulerdomain.JobPending && !job.NextAttemptAt().After(at) {
			jobs = append(jobs, job)
		}
	}
	return jobs, nil
}

func (r *scenarioJobRepository) Acquire(ctx context.Context, id schedulerdomain.JobID, owner string, expiresAt, _ time.Time) (*schedulerdomain.Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	job, exists := r.jobs[id]
	if !exists {
		return nil, ErrAsyncOperationNotFound
	}
	if err := job.Lease(owner, expiresAt); err != nil {
		return nil, err
	}
	return job, nil
}

type scenarioSagaStore struct {
	instances map[paymentworkflowdomain.ID]paymentworkflowdomain.Instance
}

func newScenarioSagaStore() *scenarioSagaStore {
	return &scenarioSagaStore{instances: make(map[paymentworkflowdomain.ID]paymentworkflowdomain.Instance)}
}
func (s *scenarioSagaStore) Save(_ context.Context, instance paymentworkflowdomain.Instance) error {
	s.instances[instance.ID] = instance
	return nil
}
func (s *scenarioSagaStore) Find(_ context.Context, id paymentworkflowdomain.ID) (paymentworkflowdomain.Instance, error) {
	instance, ok := s.instances[id]
	if !ok {
		return paymentworkflowdomain.Instance{}, ErrAsyncOperationNotFound
	}
	return instance, nil
}

type scenarioSagaPublisher struct{ jobs *scenarioJobRepository }

func (p *scenarioSagaPublisher) Publish(ctx context.Context, message paymentworkflowdomain.Message) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	job, err := schedulerdomain.NewJob(schedulerdomain.JobID(message.ID), workflowStepJobType, payload, message.ScheduledAt)
	if err != nil {
		return err
	}
	return p.jobs.Save(ctx, &job)
}

type memoryRepository struct {
	payments map[paymentdomain.ID]*paymentdomain.Payment
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		payments: make(map[paymentdomain.ID]*paymentdomain.Payment),
	}
}

func (r *memoryRepository) Save(
	ctx context.Context,
	payment *paymentdomain.Payment,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	current, exists := r.payments[payment.ID()]
	if exists && current != payment {
		return fmt.Errorf("%w: %s", ErrPaymentAlreadyExists, payment.ID())
	}

	r.payments[payment.ID()] = payment
	return nil
}

func (r *memoryRepository) FindByID(
	ctx context.Context,
	id paymentdomain.ID,
) (*paymentdomain.Payment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	payment, exists := r.payments[id]
	if !exists {
		return nil, paymentapplication.ErrPaymentNotFound
	}

	return payment, nil
}

func (r *memoryRepository) states() []paymentdomain.PaymentState {
	ids := make([]paymentdomain.ID, 0, len(r.payments))
	for id := range r.payments {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	states := make([]paymentdomain.PaymentState, 0, len(ids))
	for _, id := range ids {
		payment := r.payments[id]
		states = append(states, paymentdomain.PaymentState{
			ID:               payment.ID(),
			Amount:           payment.Amount(),
			Status:           payment.Status(),
			AuthorizedAmount: payment.AuthorizedAmount().Amount(),
			CapturedAmount:   payment.CapturedAmount().Amount(),
			RefundedAmount:   payment.RefundedAmount().Amount(),
			Version:          payment.Version(),
		})
	}

	return states
}

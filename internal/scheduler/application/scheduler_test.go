package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/scheduler/domain"
)

func TestSchedulerTick(t *testing.T) {
	now := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC)
	firstJob := newTestJob(t, "job-1")
	secondJob := newTestJob(t, "job-2")
	repository := &fakeRepository{
		jobs: []*domain.Job{firstJob, secondJob},
		acquired: map[domain.JobID]*domain.Job{
			"job-1": firstJob,
			"job-2": secondJob,
		},
	}
	dispatcher := &fakeDispatcher{}
	scheduler := newTestScheduler(t, repository, dispatcher, now)

	if err := scheduler.Tick(context.Background()); err != nil {
		t.Fatalf("Tick() error = %v", err)
	}

	if !repository.findAt.Equal(now) || repository.findLimit != 2 {
		t.Fatalf("FindExecutable() received at=%v limit=%d, want at=%v limit=2", repository.findAt, repository.findLimit, now)
	}

	wantIDs := []domain.JobID{"job-1", "job-2"}
	if !reflect.DeepEqual(repository.acquiredIDs, wantIDs) {
		t.Fatalf("acquired job ids = %v, want %v", repository.acquiredIDs, wantIDs)
	}

	if !reflect.DeepEqual(dispatcher.jobs, []*domain.Job{firstJob, secondJob}) {
		t.Fatalf("dispatched jobs = %v, want both acquired jobs", dispatcher.jobs)
	}

	if !repository.lastExpiry.Equal(now.Add(time.Minute)) {
		t.Fatalf("lease expiry = %v, want %v", repository.lastExpiry, now.Add(time.Minute))
	}
}

func TestSchedulerTick_ReturnsRepositoryError(t *testing.T) {
	wantErr := errors.New("find jobs")
	repository := &fakeRepository{findErr: wantErr}
	scheduler := newTestScheduler(t, repository, &fakeDispatcher{}, time.Unix(1, 0))

	if err := scheduler.Tick(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("Tick() error = %v, want %v", err, wantErr)
	}
}

func TestSchedulerTick_ContinuesAfterAcquisitionAndDispatchErrors(t *testing.T) {
	firstJob := newTestJob(t, "job-1")
	secondJob := newTestJob(t, "job-2")
	acquireErr := errors.New("acquire job")
	dispatchErr := errors.New("dispatch job")
	repository := &fakeRepository{
		jobs: []*domain.Job{firstJob, secondJob},
		acquired: map[domain.JobID]*domain.Job{
			"job-1": firstJob,
			"job-2": secondJob,
		},
		acquireErrs: map[domain.JobID]error{"job-1": acquireErr},
	}
	dispatcher := &fakeDispatcher{err: dispatchErr}
	scheduler := newTestScheduler(t, repository, dispatcher, time.Unix(1, 0))

	err := scheduler.Tick(context.Background())
	if !errors.Is(err, acquireErr) || !errors.Is(err, dispatchErr) {
		t.Fatalf("Tick() error = %v, want acquisition and dispatch errors", err)
	}

	if len(dispatcher.jobs) != 1 || dispatcher.jobs[0].ID() != "job-2" {
		t.Fatalf("dispatched jobs = %v, want only job-2", dispatcher.jobs)
	}
}

func TestNewScheduler_RejectsInvalidConfiguration(t *testing.T) {
	validRepository := &fakeRepository{}
	validDispatcher := &fakeDispatcher{}
	validClock := fixedClock{now: time.Unix(1, 0)}
	validConfig := Config{Owner: "scheduler-1", BatchSize: 1, LeaseDuration: time.Second}

	tests := []struct {
		name   string
		mutate func(*Config, *fakeRepository, *fakeDispatcher, *fixedClock)
		want   error
	}{
		{name: "repository", mutate: func(_ *Config, _ *fakeRepository, _ *fakeDispatcher, _ *fixedClock) {}, want: ErrInvalidRepository},
		{name: "dispatcher", mutate: func(_ *Config, _ *fakeRepository, _ *fakeDispatcher, _ *fixedClock) {}, want: ErrInvalidDispatcher},
		{name: "clock", mutate: func(_ *Config, _ *fakeRepository, _ *fakeDispatcher, _ *fixedClock) {}, want: ErrInvalidClock},
		{name: "owner", mutate: func(config *Config, _ *fakeRepository, _ *fakeDispatcher, _ *fixedClock) { config.Owner = "" }, want: ErrInvalidOwner},
		{name: "batch size", mutate: func(config *Config, _ *fakeRepository, _ *fakeDispatcher, _ *fixedClock) { config.BatchSize = 0 }, want: ErrInvalidBatchSize},
		{name: "lease duration", mutate: func(config *Config, _ *fakeRepository, _ *fakeDispatcher, _ *fixedClock) { config.LeaseDuration = 0 }, want: ErrInvalidLeaseDuration},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := validConfig
			repository := *validRepository
			dispatcher := *validDispatcher
			clock := validClock
			var repositoryArg Repository = &repository
			var dispatcherArg Dispatcher = &dispatcher
			var clockArg Clock = &clock

			switch tt.name {
			case "repository":
				repositoryArg = nil
			case "dispatcher":
				dispatcherArg = nil
			case "clock":
				clockArg = nil
			default:
				tt.mutate(&config, &repository, &dispatcher, &clock)
			}

			_, err := NewScheduler(repositoryArg, dispatcherArg, clockArg, config)
			if !errors.Is(err, tt.want) {
				t.Fatalf("NewScheduler() error = %v, want %v", err, tt.want)
			}
		})
	}
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type fakeRepository struct {
	jobs        []*domain.Job
	acquired    map[domain.JobID]*domain.Job
	acquireErrs map[domain.JobID]error
	findErr     error
	findAt      time.Time
	findLimit   int
	lastExpiry  time.Time
	acquiredIDs []domain.JobID
}

func (r *fakeRepository) FindExecutable(_ context.Context, at time.Time, limit int) ([]*domain.Job, error) {
	r.findAt = at
	r.findLimit = limit
	return r.jobs, r.findErr
}

func (r *fakeRepository) Acquire(_ context.Context, id domain.JobID, _ string, expiresAt time.Time) (*domain.Job, error) {
	r.acquiredIDs = append(r.acquiredIDs, id)
	r.lastExpiry = expiresAt
	if err := r.acquireErrs[id]; err != nil {
		return nil, err
	}
	return r.acquired[id], nil
}

type fakeDispatcher struct {
	jobs []*domain.Job
	err  error
}

func (d *fakeDispatcher) Dispatch(_ context.Context, job *domain.Job) error {
	d.jobs = append(d.jobs, job)
	return d.err
}

func newTestScheduler(t *testing.T, repository Repository, dispatcher Dispatcher, now time.Time) *Scheduler {
	t.Helper()

	scheduler, err := NewScheduler(repository, dispatcher, fixedClock{now: now}, Config{
		Owner:         "scheduler-1",
		BatchSize:     2,
		LeaseDuration: time.Minute,
	})
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}

	return scheduler
}

func newTestJob(t *testing.T, id domain.JobID) *domain.Job {
	t.Helper()

	job, err := domain.NewJob(id, "webhook.delivery", nil, time.Unix(1, 0))
	if err != nil {
		t.Fatalf("NewJob() error = %v", err)
	}

	return &job
}

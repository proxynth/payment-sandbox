package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/scheduler/domain"
)

func TestWorkerExecute_CompletesJob(t *testing.T) {
	job := leasedTestJob(t, "webhook.delivery")
	repository := &fakeWorkerRepository{}
	var receivedPayload []byte
	worker, err := NewWorker(repository, map[domain.JobType]JobHandler{
		"webhook.delivery": func(_ context.Context, payload []byte) error {
			receivedPayload = payload
			if job.Status() != domain.JobRunning {
				t.Errorf("job status during handler = %q, want %q", job.Status(), domain.JobRunning)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewWorker() error = %v", err)
	}

	if err := worker.Execute(context.Background(), job); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if job.Status() != domain.JobCompleted {
		t.Fatalf("job status = %q, want %q", job.Status(), domain.JobCompleted)
	}

	if !reflect.DeepEqual(receivedPayload, []byte("payload")) {
		t.Fatalf("handler payload = %q, want %q", receivedPayload, "payload")
	}

	if !reflect.DeepEqual(repository.savedStatuses, []domain.JobStatus{domain.JobRunning, domain.JobCompleted}) {
		t.Fatalf("saved statuses = %v, want running then completed", repository.savedStatuses)
	}
}

func TestWorkerExecute_PersistsFailedJob(t *testing.T) {
	job := leasedTestJob(t, "webhook.delivery")
	executionErr := errors.New("delivery failed")
	repository := &fakeWorkerRepository{}
	worker, err := NewWorker(repository, map[domain.JobType]JobHandler{
		"webhook.delivery": func(context.Context, []byte) error {
			return executionErr
		},
	})
	if err != nil {
		t.Fatalf("NewWorker() error = %v", err)
	}

	err = worker.Execute(context.Background(), job)
	if !errors.Is(err, executionErr) {
		t.Fatalf("Execute() error = %v, want %v", err, executionErr)
	}

	if job.Status() != domain.JobFailed {
		t.Fatalf("job status = %q, want %q", job.Status(), domain.JobFailed)
	}

	if !reflect.DeepEqual(repository.savedStatuses, []domain.JobStatus{domain.JobRunning, domain.JobFailed}) {
		t.Fatalf("saved statuses = %v, want running then failed", repository.savedStatuses)
	}
}

func TestWorkerExecute_DoesNotRunUnknownType(t *testing.T) {
	job := leasedTestJob(t, "unknown")
	called := false
	worker, err := NewWorker(&fakeWorkerRepository{}, map[domain.JobType]JobHandler{
		"webhook.delivery": func(context.Context, []byte) error {
			called = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewWorker() error = %v", err)
	}

	if err := worker.Execute(context.Background(), job); !errors.Is(err, ErrUnknownJobType) {
		t.Fatalf("Execute() error = %v, want %v", err, ErrUnknownJobType)
	}

	if called {
		t.Fatal("unknown job type was dispatched to a handler")
	}

	if job.Status() != domain.JobLeased {
		t.Fatalf("job status = %q, want %q", job.Status(), domain.JobLeased)
	}
}

func TestWorkerExecute_RejectsUnleasedJob(t *testing.T) {
	job, err := domain.NewJob("job-1", "webhook.delivery", []byte("payload"), time.Unix(1, 0))
	if err != nil {
		t.Fatalf("NewJob() error = %v", err)
	}
	worker, err := NewWorker(&fakeWorkerRepository{}, map[domain.JobType]JobHandler{
		"webhook.delivery": func(context.Context, []byte) error { return nil },
	})
	if err != nil {
		t.Fatalf("NewWorker() error = %v", err)
	}

	if err := worker.Execute(context.Background(), &job); !errors.Is(err, domain.ErrInvalidJobTransition) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidJobTransition)
	}
}

func TestWorkerExecute_ReturnsRunningPersistenceErrorWithoutExecuting(t *testing.T) {
	job := leasedTestJob(t, "webhook.delivery")
	persistenceErr := errors.New("save failed")
	repository := &fakeWorkerRepository{saveErr: persistenceErr}
	called := false
	worker, err := NewWorker(repository, map[domain.JobType]JobHandler{
		"webhook.delivery": func(context.Context, []byte) error {
			called = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewWorker() error = %v", err)
	}

	if err := worker.Execute(context.Background(), job); !errors.Is(err, persistenceErr) {
		t.Fatalf("Execute() error = %v, want %v", err, persistenceErr)
	}

	if called {
		t.Fatal("handler ran before running state was persisted")
	}
}

func TestNewWorker_RejectsInvalidConfiguration(t *testing.T) {
	if _, err := NewWorker(nil, nil); !errors.Is(err, ErrInvalidWorkerRepository) {
		t.Fatalf("NewWorker() error = %v, want %v", err, ErrInvalidWorkerRepository)
	}

	if _, err := NewWorker(&fakeWorkerRepository{}, map[domain.JobType]JobHandler{
		"webhook.delivery": nil,
	}); !errors.Is(err, ErrInvalidHandler) {
		t.Fatalf("NewWorker() error = %v, want %v", err, ErrInvalidHandler)
	}
}

type fakeWorkerRepository struct {
	saveErr       error
	savedStatuses []domain.JobStatus
}

func (r *fakeWorkerRepository) Save(_ context.Context, job *domain.Job) error {
	r.savedStatuses = append(r.savedStatuses, job.Status())
	return r.saveErr
}

func leasedTestJob(t *testing.T, jobType domain.JobType) *domain.Job {
	t.Helper()

	job, err := domain.NewJob("job-1", jobType, []byte("payload"), time.Unix(1, 0))
	if err != nil {
		t.Fatalf("NewJob() error = %v", err)
	}

	if err := job.Lease("worker-1", time.Unix(2, 0)); err != nil {
		t.Fatalf("Lease() error = %v", err)
	}

	return &job
}

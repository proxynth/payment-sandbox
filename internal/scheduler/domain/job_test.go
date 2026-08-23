package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewJob(t *testing.T) {
	scheduledAt := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	payload := []byte(`{"payment_id":"payment-1"}`)
	job, err := NewJob("job-1", "webhook.delivery", payload, scheduledAt)
	if err != nil {
		t.Fatalf("NewJob() error = %v", err)
	}

	if job.ID() != "job-1" {
		t.Errorf("ID() = %q, want %q", job.ID(), "job-1")
	}

	if job.Type() != "webhook.delivery" {
		t.Errorf("Type() = %q, want %q", job.Type(), "webhook.delivery")
	}

	if string(job.Payload()) != string(payload) {
		t.Errorf("Payload() = %q, want %q", job.Payload(), payload)
	}

	if job.Status() != JobPending {
		t.Errorf("Status() = %q, want %q", job.Status(), JobPending)
	}

	if !job.ScheduledAt().Equal(scheduledAt) || job.ScheduledAt().Location() != time.UTC {
		t.Errorf("ScheduledAt() = %v, want UTC equivalent of %v", job.ScheduledAt(), scheduledAt)
	}

	if !job.NextAttemptAt().Equal(scheduledAt) {
		t.Errorf("NextAttemptAt() = %v, want %v", job.NextAttemptAt(), scheduledAt)
	}

	if job.Attempts() != 0 {
		t.Errorf("Attempts() = %d, want 0", job.Attempts())
	}
}

func TestNewJob_RejectsInvalidInput(t *testing.T) {
	validTime := time.Unix(1, 0)
	tests := []struct {
		name      string
		id        JobID
		jobType   JobType
		scheduled time.Time
		want      error
	}{
		{name: "empty id", jobType: "webhook.delivery", scheduled: validTime, want: ErrInvalidJobID},
		{name: "empty type", id: "job-1", scheduled: validTime, want: ErrInvalidJobType},
		{name: "zero schedule", id: "job-1", jobType: "webhook.delivery", want: ErrInvalidScheduledAt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewJob(tt.id, tt.jobType, nil, tt.scheduled)
			if !errors.Is(err, tt.want) {
				t.Fatalf("NewJob() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestJobLifecycle(t *testing.T) {
	job := newTestJob(t)
	leaseExpiresAt := time.Unix(20, 0)
	retryAt := time.Unix(30, 0)

	if err := job.Lease("worker-1", leaseExpiresAt); err != nil {
		t.Fatalf("Lease() error = %v", err)
	}
	if job.Status() != JobLeased || job.LeaseOwner() != "worker-1" || !job.LeaseExpiresAt().Equal(leaseExpiresAt) {
		t.Fatalf("job should contain its lease metadata after Lease()")
	}

	if err := job.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if job.Status() != JobRunning || job.Attempts() != 1 {
		t.Fatalf("job should be running with one attempt after Start()")
	}

	if err := job.Fail(); err != nil {
		t.Fatalf("Fail() error = %v", err)
	}
	if job.Status() != JobFailed || job.LeaseOwner() != "" || !job.LeaseExpiresAt().IsZero() {
		t.Fatalf("job should be failed without a lease after Fail()")
	}

	if err := job.ScheduleRetry(retryAt); err != nil {
		t.Fatalf("ScheduleRetry() error = %v", err)
	}
	if job.Status() != JobPending || !job.NextAttemptAt().Equal(retryAt) {
		t.Fatalf("job should be pending at the requested retry time")
	}
}

func TestJobComplete_ClearsLease(t *testing.T) {
	job := newTestJob(t)
	if err := job.Lease("worker-1", time.Unix(20, 0)); err != nil {
		t.Fatalf("Lease() error = %v", err)
	}
	if err := job.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := job.Complete(); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if job.Status() != JobCompleted || job.LeaseOwner() != "" || !job.LeaseExpiresAt().IsZero() {
		t.Fatalf("completed job should not retain lease metadata")
	}
}

func TestJob_RejectsInvalidTransitionsAndMetadata(t *testing.T) {
	job := newTestJob(t)

	if err := job.Start(); !errors.Is(err, ErrInvalidJobTransition) {
		t.Fatalf("Start() error = %v, want %v", err, ErrInvalidJobTransition)
	}
	if err := job.Lease("", time.Unix(2, 0)); !errors.Is(err, ErrInvalidLeaseOwner) {
		t.Fatalf("Lease() error = %v, want %v", err, ErrInvalidLeaseOwner)
	}
	if err := job.Lease("worker-1", time.Time{}); !errors.Is(err, ErrInvalidLeaseExpiry) {
		t.Fatalf("Lease() error = %v, want %v", err, ErrInvalidLeaseExpiry)
	}
	if err := job.Lease("worker-1", time.Unix(2, 0)); err != nil {
		t.Fatalf("Lease() error = %v", err)
	}
	if err := job.Lease("worker-2", time.Unix(3, 0)); !errors.Is(err, ErrInvalidJobTransition) {
		t.Fatalf("second Lease() error = %v, want %v", err, ErrInvalidJobTransition)
	}
	if err := job.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := job.Fail(); err != nil {
		t.Fatalf("Fail() error = %v", err)
	}
	if err := job.ScheduleRetry(time.Time{}); !errors.Is(err, ErrInvalidRetryTime) {
		t.Fatalf("ScheduleRetry() error = %v, want %v", err, ErrInvalidRetryTime)
	}
}

func TestJob_CopiesPayload(t *testing.T) {
	payload := []byte("payload")
	job, err := NewJob("job-1", "generic", payload, time.Unix(1, 0))
	if err != nil {
		t.Fatalf("NewJob() error = %v", err)
	}

	payload[0] = 'X'
	readPayload := job.Payload()
	readPayload[1] = 'X'

	if string(job.Payload()) != "payload" {
		t.Fatalf("Job payload was mutated through caller-owned memory")
	}
}

func newTestJob(t *testing.T) *Job {
	t.Helper()

	job, err := NewJob("job-1", "webhook.delivery", []byte("payload"), time.Unix(1, 0))
	if err != nil {
		t.Fatalf("NewJob() error = %v", err)
	}

	return &job
}

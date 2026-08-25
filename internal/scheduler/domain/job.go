package domain

import "time"

type JobID string

type JobType string

type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobLeased    JobStatus = "leased"
	JobRunning   JobStatus = "running"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
)

type Job struct {
	id             JobID
	jobType        JobType
	payload        []byte
	scheduledAt    time.Time
	nextAttemptAt  time.Time
	status         JobStatus
	leaseOwner     string
	leaseExpiresAt time.Time
	attempts       uint64
}

func NewJob(
	id JobID,
	jobType JobType,
	payload []byte,
	scheduledAt time.Time,
) (Job, error) {
	if id == "" {
		return Job{}, ErrInvalidJobID
	}

	if jobType == "" {
		return Job{}, ErrInvalidJobType
	}

	if scheduledAt.IsZero() {
		return Job{}, ErrInvalidScheduledAt
	}

	scheduledAt = scheduledAt.UTC()

	return Job{
		id:            id,
		jobType:       jobType,
		payload:       cloneBytes(payload),
		scheduledAt:   scheduledAt,
		nextAttemptAt: scheduledAt,
		status:        JobPending,
	}, nil
}

// Restore reconstructs a persisted job without replaying its lifecycle.
// Persistence adapters are responsible for validating the stored columns
// before calling it.
func Restore(
	id JobID,
	jobType JobType,
	payload []byte,
	scheduledAt time.Time,
	nextAttemptAt time.Time,
	status JobStatus,
	leaseOwner string,
	leaseExpiresAt time.Time,
	attempts uint64,
) (Job, error) {
	if id == "" {
		return Job{}, ErrInvalidJobID
	}
	if jobType == "" {
		return Job{}, ErrInvalidJobType
	}
	if scheduledAt.IsZero() || nextAttemptAt.IsZero() {
		return Job{}, ErrInvalidScheduledAt
	}
	switch status {
	case JobPending, JobLeased, JobRunning, JobCompleted, JobFailed:
	default:
		return Job{}, ErrInvalidExecutionStatus
	}
	return Job{id: id, jobType: jobType, payload: cloneBytes(payload), scheduledAt: scheduledAt.UTC(), nextAttemptAt: nextAttemptAt.UTC(), status: status, leaseOwner: leaseOwner, leaseExpiresAt: leaseExpiresAt.UTC(), attempts: attempts}, nil
}

func (j *Job) RequeueExpired(at time.Time) bool {
	if j.status != JobLeased && j.status != JobRunning {
		return false
	}
	if j.leaseExpiresAt.IsZero() || j.leaseExpiresAt.After(at) {
		return false
	}
	j.status = JobPending
	j.leaseOwner = ""
	j.leaseExpiresAt = time.Time{}
	return true
}

func (j *Job) Lease(owner string, expiresAt time.Time) error {
	if j.status != JobPending {
		return invalidTransition(j.status, "lease")
	}

	if owner == "" {
		return ErrInvalidLeaseOwner
	}

	if expiresAt.IsZero() {
		return ErrInvalidLeaseExpiry
	}

	j.status = JobLeased
	j.leaseOwner = owner
	j.leaseExpiresAt = expiresAt.UTC()

	return nil
}

func (j *Job) Start() error {
	if j.status != JobLeased {
		return invalidTransition(j.status, "start")
	}

	j.status = JobRunning
	j.attempts++

	return nil
}

func (j *Job) Complete() error {
	if j.status != JobRunning {
		return invalidTransition(j.status, "complete")
	}

	j.status = JobCompleted
	j.clearLease()

	return nil
}

func (j *Job) Fail() error {
	if j.status != JobRunning {
		return invalidTransition(j.status, "fail")
	}

	j.status = JobFailed
	j.clearLease()

	return nil
}

func (j *Job) ScheduleRetry(at time.Time) error {
	if j.status != JobFailed {
		return invalidTransition(j.status, "schedule retry")
	}

	if at.IsZero() {
		return ErrInvalidRetryTime
	}

	j.nextAttemptAt = at.UTC()
	j.status = JobPending

	return nil
}

func (j Job) ID() JobID {
	return j.id
}

func (j Job) Type() JobType {
	return j.jobType
}

func (j Job) Payload() []byte {
	return cloneBytes(j.payload)
}

func (j Job) ScheduledAt() time.Time {
	return j.scheduledAt
}

func (j Job) NextAttemptAt() time.Time {
	return j.nextAttemptAt
}

func (j Job) Status() JobStatus {
	return j.status
}

func (j Job) LeaseOwner() string {
	return j.leaseOwner
}

func (j Job) LeaseExpiresAt() time.Time {
	return j.leaseExpiresAt
}

func (j Job) Attempts() uint64 {
	return j.attempts
}

func (j *Job) clearLease() {
	j.leaseOwner = ""
	j.leaseExpiresAt = time.Time{}
}

func cloneBytes(value []byte) []byte {
	return append([]byte(nil), value...)
}

func invalidTransition(status JobStatus, operation string) error {
	return &transitionError{status: status, operation: operation}
}

type transitionError struct {
	status    JobStatus
	operation string
}

func (e *transitionError) Error() string {
	return ErrInvalidJobTransition.Error() + ": " + e.operation + " job in status " + string(e.status)
}

func (e *transitionError) Unwrap() error {
	return ErrInvalidJobTransition
}

package domain

import "time"

// JobSnapshot is an immutable, restorable record of a scheduler job lifecycle
// state. Unlike a status-only audit row, it retains the dispatch type and
// payload needed to reconstruct the job without consulting scheduler_jobs.
type JobSnapshot struct {
	ID              JobID
	Type            JobType
	Payload         []byte
	ScheduledAt     time.Time
	NextAttemptAt   time.Time
	Status          JobStatus
	LeaseOwner      string
	LeaseExpiresAt  time.Time
	Attempts        uint64
	AggregateID     string
	CausationID     string
	RuntimeSequence uint64
}

func NewJobSnapshot(job *Job) JobSnapshot {
	return JobSnapshot{
		ID: job.ID(), Type: job.Type(), Payload: job.Payload(), ScheduledAt: job.ScheduledAt(),
		NextAttemptAt: job.NextAttemptAt(), Status: job.Status(), LeaseOwner: job.LeaseOwner(),
		LeaseExpiresAt: job.LeaseExpiresAt(), Attempts: job.Attempts(), AggregateID: job.AggregateID(), CausationID: job.CausationID(),
	}
}

// Restore recreates the exact job state represented by the snapshot.
func (s JobSnapshot) Restore() (*Job, error) {
	job, err := Restore(s.ID, s.Type, s.Payload, s.ScheduledAt, s.NextAttemptAt, s.Status, s.LeaseOwner, s.LeaseExpiresAt, s.Attempts, JobMetadata{AggregateID: s.AggregateID, CausationID: s.CausationID})
	if err != nil {
		return nil, err
	}
	return &job, nil
}

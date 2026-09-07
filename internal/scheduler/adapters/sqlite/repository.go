package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	persistencesqlite "proxynth/payment-sandbox/internal/platform/persistence/sqlite"
	"proxynth/payment-sandbox/internal/scheduler/domain"
)

type executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Repository struct{ db executor }

const sqliteTimestampLayout = "2006-01-02T15:04:05.000000000Z"

func formatTimestamp(at time.Time) string { return at.UTC().Format(sqliteTimestampLayout) }

func NewRepository(db executor) *Repository { return &Repository{db: db} }

func (r *Repository) Save(ctx context.Context, job *domain.Job) error {
	if job == nil {
		return fmt.Errorf("nil scheduler job")
	}
	leaseExpires := ""
	if !job.LeaseExpiresAt().IsZero() {
		leaseExpires = formatTimestamp(job.LeaseExpiresAt())
	}
	exec := r.db
	if tx := persistencesqlite.TxFromContext(ctx); tx != nil {
		exec = tx
	}
	_, err := exec.ExecContext(ctx, `
		INSERT INTO scheduler_jobs(id,type,payload,scheduled_at,next_attempt_at,status,lease_owner,lease_expires_at,attempts)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT(id) DO UPDATE SET type=excluded.type,payload=excluded.payload,
		 scheduled_at=excluded.scheduled_at,next_attempt_at=excluded.next_attempt_at,status=excluded.status,
		 lease_owner=excluded.lease_owner,lease_expires_at=excluded.lease_expires_at,attempts=excluded.attempts
		 WHERE scheduler_jobs.status <> $10`,
		job.ID(), job.Type(), job.Payload(), formatTimestamp(job.ScheduledAt()), formatTimestamp(job.NextAttemptAt()), job.Status(), job.LeaseOwner(), leaseExpires, job.Attempts(), domain.JobCompleted)
	if err != nil {
		return fmt.Errorf("save scheduler job %q: %w", job.ID(), err)
	}
	return nil
}

func (r *Repository) FindExecutable(ctx context.Context, at time.Time, limit int) ([]*domain.Job, error) {
	exec := r.db
	if tx := persistencesqlite.TxFromContext(ctx); tx != nil {
		exec = tx
	}
	rows, err := exec.QueryContext(ctx, `SELECT id,type,payload,scheduled_at,next_attempt_at,status,lease_owner,lease_expires_at,attempts FROM scheduler_jobs WHERE ((status IN ($1,$2) AND next_attempt_at <= $3) OR (status IN ($5,$6) AND lease_expires_at <> '' AND lease_expires_at <= $3)) ORDER BY next_attempt_at,id LIMIT $4`, domain.JobPending, domain.JobFailed, formatTimestamp(at), limit, domain.JobLeased, domain.JobRunning)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var jobs []*domain.Job
	for rows.Next() {
		job, err := scan(rows)
		if err != nil {
			return nil, err
		}
		if job.Status() == domain.JobFailed {
			if err := job.ScheduleRetry(at); err != nil {
				return nil, err
			}
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *Repository) Acquire(ctx context.Context, id domain.JobID, owner string, expiresAt, leaseCheckAt time.Time) (*domain.Job, error) {
	exec := r.db
	if tx := persistencesqlite.TxFromContext(ctx); tx != nil {
		exec = tx
	}
	result, err := exec.ExecContext(ctx, `UPDATE scheduler_jobs SET status=$1, lease_owner=$2, lease_expires_at=$3 WHERE id=$4 AND (status=$5 OR (status IN ($6,$7) AND lease_expires_at <> '' AND lease_expires_at <= $8))`, domain.JobLeased, owner, formatTimestamp(expiresAt), id, domain.JobPending, domain.JobLeased, domain.JobRunning, formatTimestamp(leaseCheckAt))
	if err != nil {
		return nil, fmt.Errorf("acquire job %q: %w", id, err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("acquire job %q rows affected: %w", id, err)
	}
	if changed == 0 {
		return nil, fmt.Errorf("acquire job %q: %w", id, domain.ErrInvalidJobTransition)
	}
	return r.find(ctx, id)
}

func (r *Repository) find(ctx context.Context, id domain.JobID) (*domain.Job, error) {
	exec := r.db
	if tx := persistencesqlite.TxFromContext(ctx); tx != nil {
		exec = tx
	}
	row := exec.QueryRowContext(ctx, `SELECT id,type,payload,scheduled_at,next_attempt_at,status,lease_owner,lease_expires_at,attempts FROM scheduler_jobs WHERE id=$1`, id)
	return scan(row)
}

type scanner interface{ Scan(...any) error }

func scan(row scanner) (*domain.Job, error) {
	var id, jobType, scheduled, nextAttempt, status, owner, leaseExpires string
	var payload []byte
	var attempts uint64
	if err := row.Scan(&id, &jobType, &payload, &scheduled, &nextAttempt, &status, &owner, &leaseExpires, &attempts); err != nil {
		return nil, err
	}
	scheduledAt, err := time.Parse(time.RFC3339Nano, scheduled)
	if err != nil {
		return nil, err
	}
	nextAt, err := time.Parse(time.RFC3339Nano, nextAttempt)
	if err != nil {
		return nil, err
	}
	leaseAt := time.Time{}
	if leaseExpires != "" {
		leaseAt, err = time.Parse(time.RFC3339Nano, leaseExpires)
		if err != nil {
			return nil, err
		}
	}
	job, err := domain.Restore(domain.JobID(id), domain.JobType(jobType), payload, scheduledAt, nextAt, domain.JobStatus(status), owner, leaseAt, attempts)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

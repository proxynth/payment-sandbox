package sqlite

import (
	"context"
	"crypto/sha256"
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
	result, err := exec.ExecContext(ctx, `
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
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("save scheduler job %q rows affected: %w", job.ID(), err)
	}
	if changed > 0 {
		if err := appendAudit(ctx, exec, job); err != nil {
			return err
		}
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
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *Repository) Acquire(ctx context.Context, id domain.JobID, owner string, expiresAt, leaseCheckAt time.Time) (*domain.Job, error) {
	exec := r.db
	if tx := persistencesqlite.TxFromContext(ctx); tx != nil {
		exec = tx
	}
	result, err := exec.ExecContext(ctx, `UPDATE scheduler_jobs SET status=$1, lease_owner=$2, lease_expires_at=$3 WHERE id=$4 AND (status IN ($5,$6) OR (status IN ($7,$8) AND lease_expires_at <> '' AND lease_expires_at <= $9))`, domain.JobLeased, owner, formatTimestamp(expiresAt), id, domain.JobPending, domain.JobFailed, domain.JobLeased, domain.JobRunning, formatTimestamp(leaseCheckAt))
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
	job, err := r.find(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := appendAudit(ctx, exec, job); err != nil {
		return nil, err
	}
	return job, nil
}

func appendAudit(ctx context.Context, exec executor, job *domain.Job) error {
	leaseExpires := ""
	if !job.LeaseExpiresAt().IsZero() {
		leaseExpires = formatTimestamp(job.LeaseExpiresAt())
	}
	snapshot := fmt.Sprintf("%s|%s|%x|%s|%d|%s|%s|%s|%s", job.ID(), job.Type(), job.Payload(), job.Status(), job.Attempts(), formatTimestamp(job.ScheduledAt()), formatTimestamp(job.NextAttemptAt()), job.LeaseOwner(), leaseExpires)
	hash := sha256.Sum256([]byte(snapshot))
	_, err := exec.ExecContext(ctx, `INSERT INTO scheduler_job_audit(id,job_id,job_type,payload,status,attempts,scheduled_at,next_attempt_at,lease_owner,lease_expires_at) VALUES (?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO NOTHING`, fmt.Sprintf("%x", hash[:]), job.ID(), job.Type(), job.Payload(), job.Status(), job.Attempts(), formatTimestamp(job.ScheduledAt()), formatTimestamp(job.NextAttemptAt()), job.LeaseOwner(), leaseExpires)
	if err != nil {
		return fmt.Errorf("append scheduler audit for job %q: %w", job.ID(), err)
	}
	return nil
}

// ListAudit returns restorable lifecycle snapshots in deterministic order.
func (r *Repository) ListAudit(ctx context.Context, id domain.JobID) ([]domain.JobSnapshot, error) {
	exec := r.db
	if tx := persistencesqlite.TxFromContext(ctx); tx != nil {
		exec = tx
	}
	rows, err := exec.QueryContext(ctx, `SELECT job_type,payload,status,attempts,scheduled_at,next_attempt_at,lease_owner,lease_expires_at FROM scheduler_job_audit WHERE job_id = ? ORDER BY attempts, next_attempt_at, CASE status WHEN 'pending' THEN 1 WHEN 'leased' THEN 2 WHEN 'running' THEN 3 WHEN 'failed' THEN 4 WHEN 'completed' THEN 5 END, id`, id)
	if err != nil {
		return nil, fmt.Errorf("list scheduler audit for job %q: %w", id, err)
	}
	defer rows.Close()
	snapshots := make([]domain.JobSnapshot, 0)
	for rows.Next() {
		var jobType, status, scheduledAt, nextAttemptAt, leaseOwner, leaseExpiresAt string
		var payload []byte
		var attempts uint64
		if err := rows.Scan(&jobType, &payload, &status, &attempts, &scheduledAt, &nextAttemptAt, &leaseOwner, &leaseExpiresAt); err != nil {
			return nil, fmt.Errorf("scan scheduler audit for job %q: %w", id, err)
		}
		scheduled, err := time.Parse(time.RFC3339Nano, scheduledAt)
		if err != nil {
			return nil, err
		}
		next, err := time.Parse(time.RFC3339Nano, nextAttemptAt)
		if err != nil {
			return nil, err
		}
		lease := time.Time{}
		if leaseExpiresAt != "" {
			lease, err = time.Parse(time.RFC3339Nano, leaseExpiresAt)
			if err != nil {
				return nil, err
			}
		}
		snapshots = append(snapshots, domain.JobSnapshot{ID: id, Type: domain.JobType(jobType), Payload: append([]byte(nil), payload...), Status: domain.JobStatus(status), Attempts: attempts, ScheduledAt: scheduled, NextAttemptAt: next, LeaseOwner: leaseOwner, LeaseExpiresAt: lease})
	}
	return snapshots, rows.Err()
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

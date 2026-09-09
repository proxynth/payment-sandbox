-- +goose Up

CREATE TABLE scheduler_job_audit (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL,
    status TEXT NOT NULL,
    attempts INTEGER NOT NULL,
    scheduled_at TEXT NOT NULL,
    next_attempt_at TEXT NOT NULL,
    lease_owner TEXT NOT NULL,
    lease_expires_at TEXT NOT NULL
);
CREATE INDEX scheduler_job_audit_job_order ON scheduler_job_audit(job_id, attempts, status, id);

-- +goose Down

DROP INDEX scheduler_job_audit_job_order;
DROP TABLE scheduler_job_audit;

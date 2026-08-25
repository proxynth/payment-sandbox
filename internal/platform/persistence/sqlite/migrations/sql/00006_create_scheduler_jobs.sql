-- +goose Up
CREATE TABLE scheduler_jobs (
    id TEXT PRIMARY KEY NOT NULL,
    type TEXT NOT NULL,
    payload BLOB NOT NULL,
    scheduled_at TEXT NOT NULL,
    next_attempt_at TEXT NOT NULL,
    status TEXT NOT NULL,
    lease_owner TEXT NOT NULL DEFAULT '',
    lease_expires_at TEXT NOT NULL DEFAULT '',
    attempts INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX scheduler_jobs_executable ON scheduler_jobs (status, next_attempt_at);

-- +goose Down
DROP INDEX scheduler_jobs_executable;
DROP TABLE scheduler_jobs;

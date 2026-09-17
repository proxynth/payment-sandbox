-- +goose Up
ALTER TABLE scheduler_jobs ADD COLUMN lease_generation INTEGER NOT NULL DEFAULT 0;
ALTER TABLE scheduler_job_audit ADD COLUMN lease_generation INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE scheduler_job_audit DROP COLUMN lease_generation;
ALTER TABLE scheduler_jobs DROP COLUMN lease_generation;

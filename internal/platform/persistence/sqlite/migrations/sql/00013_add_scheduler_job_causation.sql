-- +goose Up

ALTER TABLE scheduler_jobs ADD COLUMN aggregate_id TEXT NOT NULL DEFAULT '';
ALTER TABLE scheduler_jobs ADD COLUMN causation_id TEXT NOT NULL DEFAULT '';
ALTER TABLE scheduler_job_audit ADD COLUMN aggregate_id TEXT NOT NULL DEFAULT '';
ALTER TABLE scheduler_job_audit ADD COLUMN causation_id TEXT NOT NULL DEFAULT '';
CREATE INDEX scheduler_jobs_aggregate ON scheduler_jobs(aggregate_id, causation_id);

-- +goose Down

DROP INDEX scheduler_jobs_aggregate;
ALTER TABLE scheduler_job_audit DROP COLUMN causation_id;
ALTER TABLE scheduler_job_audit DROP COLUMN aggregate_id;
ALTER TABLE scheduler_jobs DROP COLUMN causation_id;
ALTER TABLE scheduler_jobs DROP COLUMN aggregate_id;

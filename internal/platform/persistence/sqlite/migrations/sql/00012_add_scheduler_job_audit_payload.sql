-- +goose Up

ALTER TABLE scheduler_job_audit ADD COLUMN job_type TEXT NOT NULL DEFAULT '';
ALTER TABLE scheduler_job_audit ADD COLUMN payload BLOB NOT NULL DEFAULT X'';

UPDATE scheduler_job_audit
SET job_type = (SELECT type FROM scheduler_jobs WHERE scheduler_jobs.id = scheduler_job_audit.job_id),
    payload = (SELECT payload FROM scheduler_jobs WHERE scheduler_jobs.id = scheduler_job_audit.job_id)
WHERE EXISTS (SELECT 1 FROM scheduler_jobs WHERE scheduler_jobs.id = scheduler_job_audit.job_id);

-- +goose Down

ALTER TABLE scheduler_job_audit DROP COLUMN payload;
ALTER TABLE scheduler_job_audit DROP COLUMN job_type;

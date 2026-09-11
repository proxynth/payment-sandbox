-- +goose Up

CREATE TABLE runtime_sequence (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    value INTEGER NOT NULL
);
INSERT INTO runtime_sequence(id, value) VALUES (1, 0);

ALTER TABLE event_log ADD COLUMN runtime_sequence INTEGER NOT NULL DEFAULT 0;
ALTER TABLE scheduler_job_audit ADD COLUMN runtime_sequence INTEGER NOT NULL DEFAULT 0;
ALTER TABLE webhook_delivery_audit ADD COLUMN runtime_sequence INTEGER NOT NULL DEFAULT 0;

-- +goose Down

ALTER TABLE webhook_delivery_audit DROP COLUMN runtime_sequence;
ALTER TABLE scheduler_job_audit DROP COLUMN runtime_sequence;
ALTER TABLE event_log DROP COLUMN runtime_sequence;
DROP TABLE runtime_sequence;

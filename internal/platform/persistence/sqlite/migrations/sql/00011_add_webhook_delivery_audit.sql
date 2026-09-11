-- +goose Up

CREATE TABLE webhook_delivery_audit (
    job_id TEXT NOT NULL,
    attempt INTEGER NOT NULL,
    endpoint_id TEXT NOT NULL,
    correlation_id TEXT NOT NULL,
    causation_id TEXT NOT NULL,
    outcome TEXT NOT NULL,
    http_status INTEGER NOT NULL,
    error TEXT NOT NULL,
    PRIMARY KEY (job_id, attempt)
);
CREATE INDEX webhook_delivery_audit_endpoint_order ON webhook_delivery_audit(endpoint_id, job_id, attempt);

-- +goose Down

DROP INDEX webhook_delivery_audit_endpoint_order;
DROP TABLE webhook_delivery_audit;

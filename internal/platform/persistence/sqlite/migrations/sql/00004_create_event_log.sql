-- +goose Up

CREATE TABLE event_log (
    id TEXT PRIMARY KEY,
    aggregate_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    occurred_at TEXT NOT NULL,
    aggregate_version INTEGER NOT NULL,
    correlation_id TEXT NOT NULL DEFAULT '',
    causation_id TEXT NOT NULL DEFAULT ''
);

CREATE INDEX event_log_aggregate_order
    ON event_log (aggregate_id, occurred_at, aggregate_version, id);

-- +goose Down

DROP INDEX event_log_aggregate_order;
DROP TABLE event_log;

-- +goose Up

CREATE TABLE idempotency_records (
    scope TEXT NOT NULL,
    key TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    status TEXT NOT NULL,
    response_status INTEGER NOT NULL DEFAULT 0,
    response_body BLOB NOT NULL DEFAULT '',
    PRIMARY KEY (scope, key)
);

-- +goose Down

DROP TABLE idempotency_records;

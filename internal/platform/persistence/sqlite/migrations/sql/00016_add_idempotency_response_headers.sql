-- +goose Up
ALTER TABLE idempotency_records ADD COLUMN response_headers TEXT NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE idempotency_records DROP COLUMN response_headers;

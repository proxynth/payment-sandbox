-- +goose Up

ALTER TABLE event_log ADD COLUMN payload BLOB NOT NULL DEFAULT X'';

-- +goose Down

ALTER TABLE event_log DROP COLUMN payload;

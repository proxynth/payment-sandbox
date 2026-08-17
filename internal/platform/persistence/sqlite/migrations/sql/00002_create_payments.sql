-- +goose Up

CREATE TABLE payments (
    id TEXT PRIMARY KEY,
    status TEXT NOT NULL,
    version INTEGER NOT NULL
);

-- +goose Down

DROP TABLE payments
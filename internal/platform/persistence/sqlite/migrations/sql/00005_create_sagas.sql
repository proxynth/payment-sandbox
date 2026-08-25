-- +goose Up
CREATE TABLE saga_instances (
    id TEXT PRIMARY KEY NOT NULL,
    payment_id TEXT NOT NULL,
    status TEXT NOT NULL,
    current_step TEXT NOT NULL,
    completed_steps TEXT NOT NULL,
    compensation_steps TEXT NOT NULL,
    seed INTEGER NOT NULL,
    version INTEGER NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX saga_instances_payment ON saga_instances (payment_id);

-- +goose Down
DROP INDEX saga_instances_payment;
DROP TABLE saga_instances;

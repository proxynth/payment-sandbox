-- +goose Up

CREATE TABLE runtime_state (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE webhook_endpoints (id TEXT PRIMARY KEY, url TEXT NOT NULL);
CREATE TABLE replay_scenarios (id TEXT PRIMARY KEY, payload BLOB NOT NULL);

-- +goose Down

DROP TABLE replay_scenarios;
DROP TABLE webhook_endpoints;
DROP TABLE runtime_state;

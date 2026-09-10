package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type RuntimeStateStore struct{ db *sql.DB }

func NewRuntimeStateStore(db *sql.DB) *RuntimeStateStore { return &RuntimeStateStore{db: db} }

func (s *RuntimeStateStore) Load(ctx context.Context, key string) (time.Time, bool, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM runtime_state WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("load runtime state %q: %w", key, err)
	}
	at, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("parse runtime state %q: %w", key, err)
	}
	return at.UTC(), true, nil
}

func (s *RuntimeStateStore) Save(ctx context.Context, key string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO runtime_state(key,value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, at.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save runtime state %q: %w", key, err)
	}
	return nil
}

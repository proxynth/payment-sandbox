package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"proxynth/payment-sandbox/internal/platform/config"

	_ "modernc.org/sqlite"
)

const driverName = "sqlite"

func Open(ctx context.Context, cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open(driverName, cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	success := false
	defer func() {
		if !success {
			_ = db.Close()
		}
	}()

	// Payment Sandbox is primarily a local application.
	// A single connection avoids surprising behavior with
	// connection-local PRAGMs and is sufficient for now.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := configure(ctx, db, cfg); err != nil {
		return nil, fmt.Errorf("configure sqlite database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}

	success = true

	return db, nil
}

func configure(ctx context.Context, db *sql.DB, cfg config.DatabaseConfig) error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		fmt.Sprintf(
			"PRAGMA busy_timeout = %d;",
			cfg.BusyTimeout.Milliseconds(),
		),
	}

	for _, pragma := range pragmas {
		if err := executePragma(ctx, db, pragma); err != nil {
			return err
		}
	}

	return nil
}

func executePragma(
	ctx context.Context,
	db *sql.DB,
	pragma string,
) error {
	if _, err := db.ExecContext(ctx, pragma); err != nil {
		return fmt.Errorf("execute %q: %w", pragma, err)
	}

	return nil
}

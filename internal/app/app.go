package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"proxynth/payment-sandbox/internal/platform/config"
	"proxynth/payment-sandbox/internal/platform/logging"
	"proxynth/payment-sandbox/internal/platform/persistence/sqlite"
	"proxynth/payment-sandbox/internal/platform/persistence/sqlite/migrations"
)

func Run() error {
	return run(context.Background(), os.Stdout)
}

func run(ctx context.Context, output io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	logger, err := logging.New(output, cfg.Logging)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	database, err := sqlite.Open(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	if err := migrations.Up(database); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	logger.Info(
		"payment sandbox starting",
		"database", cfg.Database.Path,
		"log_level", cfg.Logging.Level,
		"log_format", cfg.Logging.Format,
	)

	return nil
}

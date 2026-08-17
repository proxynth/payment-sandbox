package config

import (
	"fmt"
	"os"

	"proxynth/payment-sandbox/internal/platform/logging"
)

const (
	envLogLevel            = "PAYMENT_SANDBOX_LOG_LEVEL"
	envLogFormat           = "PAYMENT_SANDBOX_LOG_FORMAT"
	envDatabasePath        = "PAYMENT_SANDBOX_DATABASE_PATH"
	envDatabaseBusyTimeout = "PAYMENT_SANDBOX_DATABASE_BUSY_TIMEOUT"
)

func Load() (Config, error) {
	cfg := Default()

	if value := os.Getenv(envLogLevel); value != "" {
		cfg.Logging.Level = logging.Level(value)
	}

	if value := os.Getenv(envLogFormat); value != "" {
		cfg.Logging.Format = logging.Format(value)
	}

	if err := validate(cfg); err != nil {
		return Config{}, fmt.Errorf("validate configuration: %w", err)
	}

	return cfg, nil
}

func validate(cfg Config) error {
	if !cfg.Logging.Level.Valid() {
		return fmt.Errorf("unsupported log level %q", cfg.Logging.Level)
	}

	if !cfg.Logging.Format.Valid() {
		return fmt.Errorf("unsupported log format %q", cfg.Logging.Format)
	}

	return nil
}

package config

import (
	"fmt"
	"os"
	"time"

	"proxynth/payment-sandbox/internal/platform/logging"
)

const (
	envLogLevel            = "PAYMENT_SANDBOX_LOG_LEVEL"
	envLogFormat           = "PAYMENT_SANDBOX_LOG_FORMAT"
	envDatabasePath        = "PAYMENT_SANDBOX_DATABASE_PATH"
	envDatabaseBusyTimeout = "PAYMENT_SANDBOX_DATABASE_BUSY_TIMEOUT"
	envHTTPAddress         = "PAYMENT_SANDBOX_HTTP_ADDRESS"
	envAdminToken          = "PAYMENT_SANDBOX_ADMIN_TOKEN"
)

func Load() (Config, error) {
	cfg := Default()

	if value := os.Getenv(envLogLevel); value != "" {
		cfg.Logging.Level = logging.Level(value)
	}

	if value := os.Getenv(envLogFormat); value != "" {
		cfg.Logging.Format = logging.Format(value)
	}

	if value := os.Getenv(envDatabasePath); value != "" {
		cfg.Database.Path = value
	}

	if value := os.Getenv(envDatabaseBusyTimeout); value != "" {
		busyTimeout, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf(
				"parse database busy timeout %q: %w",
				value,
				err,
			)
		}
		cfg.Database.BusyTimeout = busyTimeout
	}

	if value := os.Getenv(envHTTPAddress); value != "" {
		cfg.HTTP.Address = value
	}

	if value := os.Getenv(envAdminToken); value != "" {
		cfg.Admin.Token = value
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

	if cfg.HTTP.Address == "" {
		return fmt.Errorf("HTTP address must not be empty")
	}

	return nil
}

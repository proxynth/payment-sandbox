package config

import (
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/platform/logging"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("PAYMENT_SANDBOX_LOG_LEVEL", "")
	t.Setenv("PAYMENT_SANDBOX_LOG_FORMAT", "")
	t.Setenv("PAYMENT_SANDBOX_DATABASE_PATH", "")
	t.Setenv("PAYMENT_SANDBOX_DATABASE_BUSY_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Logging.Level != logging.LevelInfo {
		t.Errorf(
			"Logging.Level = %q, want %q",
			cfg.Logging.Level,
			logging.LevelInfo,
		)
	}

	if cfg.Logging.Format != logging.FormatText {
		t.Errorf(
			"Logging.Format = %q, want %q",
			cfg.Logging.Format,
			logging.FormatText)
	}
}

func TestLoad_OverridesDatabaseConfiguration(t *testing.T) {
	t.Setenv("PAYMENT_SANDBOX_DATABASE_PATH", "/tmp/payment-sandbox-test.db")
	t.Setenv("PAYMENT_SANDBOX_DATABASE_BUSY_TIMEOUT", "250ms")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Database.Path != "/tmp/payment-sandbox-test.db" {
		t.Errorf("Database.Path = %q, want %q", cfg.Database.Path, "/tmp/payment-sandbox-test.db")
	}
	if cfg.Database.BusyTimeout != 250*time.Millisecond {
		t.Errorf("Database.BusyTimeout = %v, want %v", cfg.Database.BusyTimeout, 250*time.Millisecond)
	}
}

func TestLoad_RejectsInvalidDatabaseBusyTimeout(t *testing.T) {
	t.Setenv("PAYMENT_SANDBOX_DATABASE_BUSY_TIMEOUT", "not-a-duration")

	_, err := Load()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoad_OverridesLoggingConfiguration(t *testing.T) {
	t.Setenv("PAYMENT_SANDBOX_LOG_LEVEL", "debug")
	t.Setenv("PAYMENT_SANDBOX_LOG_FORMAT", "json")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Logging.Level != logging.LevelDebug {
		t.Errorf(
			"Logging.Level = %q, want %q",
			cfg.Logging.Level,
			logging.LevelDebug,
		)
	}

	if cfg.Logging.Format != logging.FormatJSON {
		t.Errorf(
			"Logging.Format = %q, want %q",
			cfg.Logging.Format,
			logging.FormatJSON,
		)
	}
}

func TestLoad_RejectsInvalidLogLevel(t *testing.T) {
	t.Setenv("PAYMENT_SANDBOX_LOG_LEVEL", "verbose")

	_, err := Load()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

package config

import (
	"testing"

	"proxynth/payment-sandbox/internal/platform/logging"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("PAYMENT_SANDBOX_LOG_LEVEL", "")
	t.Setenv("PAYMENT_SANDBOX_LOG_FORMAT", "")

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

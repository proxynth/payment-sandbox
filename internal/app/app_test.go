package app

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_StartWithValidConfiguration(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "payment-sandbox.db")

	t.Setenv("PAYMENT_SANDBOX_LOG_LEVEL", "info")
	t.Setenv("PAYMENT_SANDBOX_LOG_FORMAT", "text")
	t.Setenv("PAYMENT_SANDBOX_DATABASE_PATH", dbPath)
	t.Setenv("PAYMENT_SANDBOX_DATABASE_BUSY_TIMEOUT", "5s")

	var output bytes.Buffer

	err := run(context.Background(), &output)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if !strings.Contains(output.String(), "payment sandbox starting") {
		t.Errorf(
			"output = %q, expected startup log",
			output.String(),
		)
	}
}

func TestRun_ReturnsConfigurationError(t *testing.T) {
	t.Setenv("PAYMENT_SANDBOX_LOG_LEVEL", "invalid")

	var output bytes.Buffer

	err := run(context.Background(), &output)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "load configuration") {
		t.Errorf(
			"error = %q, expected configuration context",
			err,
		)
	}
}

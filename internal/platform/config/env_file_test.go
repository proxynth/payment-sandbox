package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFileDoesNotOverrideExistingEnvironment(t *testing.T) {
	t.Setenv("PAYMENT_SANDBOX_TEST_FROM_FILE", "existing")

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("PAYMENT_SANDBOX_TEST_FROM_FILE=from-file\nPAYMENT_SANDBOX_TEST_NEW=value\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	if err := LoadEnvFile(path); err != nil {
		t.Fatalf("LoadEnvFile() error = %v", err)
	}
	if got := os.Getenv("PAYMENT_SANDBOX_TEST_FROM_FILE"); got != "existing" {
		t.Fatalf("existing variable = %q, want %q", got, "existing")
	}
	if got := os.Getenv("PAYMENT_SANDBOX_TEST_NEW"); got != "value" {
		t.Fatalf("new variable = %q, want %q", got, "value")
	}
}

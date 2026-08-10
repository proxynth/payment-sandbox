package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestNew_JSONFormat(t *testing.T) {
	var buf bytes.Buffer

	logger, err := New(&buf, Config{
		Format: FormatJSON,
		Level:  LevelInfo,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.Info(
		"payment created",
		"payment_id", "pay_123",
	)

	var entry map[string]any

	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON log, got %q: %v", buf.Bytes(), err)
	}

	if got := entry["msg"]; got != "payment created" {
		t.Errorf("msg = %q, want %q", got, "payment created")
	}

	if got := entry["payment_id"]; got != "pay_123" {
		t.Errorf("payment_id = %q, want %q", got, "pay_123")
	}

	if got := entry["level"]; got != "INFO" {
		t.Errorf("level = %q, want %q", got, "INFO")
	}
}

func TestNew_TextFormat(t *testing.T) {
	var buf bytes.Buffer

	logger, err := New(&buf, Config{
		Format: FormatText,
		Level:  LevelInfo,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.Info("application started")

	got := buf.String()

	if !strings.Contains(got, "msg=\"application started\"") {
		t.Errorf("log output = %q, expected message", got)
	}

	if !strings.Contains(got, "level=INFO") {
		t.Errorf("log output = %q, expected INFO level", got)
	}
}

func TestNew_FiltersLogsBelowConfiguredLevel(t *testing.T) {
	var buf bytes.Buffer

	logger, err := New(&buf, Config{
		Format: FormatJSON,
		Level:  LevelWarn,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.Info("should not appear")
	logger.Warn("should appear")

	got := buf.String()

	if strings.Contains(got, "should not appear") {
		t.Errorf("unexpected info log: %q", got)
	}

	if !strings.Contains(got, "should appear") {
		t.Errorf("expected warn log: %q", got)
	}
}

func TestNew_RejectsUnsupportedFormat(t *testing.T) {
	_, err := New(io.Discard, Config{
		Format: Format("xml"),
		Level:  LevelInfo,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNew_RejectsUnsupportedLevel(t *testing.T) {
	_, err := New(io.Discard, Config{
		Format: FormatText,
		Level:  Level("trace"),
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

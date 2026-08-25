package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"proxynth/payment-sandbox/internal/platform/config"
	"proxynth/payment-sandbox/internal/platform/persistence/sqlite"
	"proxynth/payment-sandbox/internal/platform/persistence/sqlite/migrations"
)

func TestRun_StartWithValidConfiguration(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "payment-sandbox.db")

	t.Setenv("PAYMENT_SANDBOX_LOG_LEVEL", "info")
	t.Setenv("PAYMENT_SANDBOX_LOG_FORMAT", "text")
	t.Setenv("PAYMENT_SANDBOX_DATABASE_PATH", dbPath)
	t.Setenv("PAYMENT_SANDBOX_DATABASE_BUSY_TIMEOUT", "5s")
	t.Setenv("PAYMENT_SANDBOX_HTTP_ADDRESS", "127.0.0.1:0")

	var output bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, &output)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	err := <-errCh
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

func TestCompose_RegistersHealthAndApplicationRoutes(t *testing.T) {
	cfg := config.Default()
	cfg.Database.Path = filepath.Join(t.TempDir(), "payment-sandbox.db")
	cfg.HTTP.Address = "127.0.0.1:0"

	database, err := sqlite.Open(context.Background(), cfg.Database)
	if err != nil {
		t.Fatalf("sqlite.Open() error = %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			t.Errorf("database.Close() error = %v", err)
		}
	}()
	if err := migrations.Up(database); err != nil {
		t.Fatalf("migrations.Up() error = %v", err)
	}

	application, err := compose(cfg, database)
	if err != nil {
		t.Fatalf("compose() error = %v", err)
	}

	server := httptest.NewServer(application.server.Handler())
	defer server.Close()

	for _, path := range []string{"/health/live", "/health/ready", "/webhook-endpoints", "/admin/providers"} {
		request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+path, nil)
		if err != nil {
			t.Fatalf("create GET %s request error = %v", path, err)
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatalf("GET %s error = %v", path, err)
		}
		if err := response.Body.Close(); err != nil {
			t.Errorf("close GET %s response body error = %v", path, err)
		}
		if response.StatusCode != http.StatusOK {
			t.Errorf("GET %s status = %d, want %d", path, response.StatusCode, http.StatusOK)
		}
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

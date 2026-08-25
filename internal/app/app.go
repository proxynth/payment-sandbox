package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	administrationhttp "proxynth/payment-sandbox/internal/administration/adapters/http"
	"proxynth/payment-sandbox/internal/api"
	paymenthttp "proxynth/payment-sandbox/internal/payment/adapters/http"
	paymentsqlite "proxynth/payment-sandbox/internal/payment/adapters/sqlite"
	"proxynth/payment-sandbox/internal/platform/clock"
	"proxynth/payment-sandbox/internal/platform/config"
	"proxynth/payment-sandbox/internal/platform/logging"
	"proxynth/payment-sandbox/internal/platform/persistence/sqlite"
	"proxynth/payment-sandbox/internal/platform/persistence/sqlite/migrations"
	"proxynth/payment-sandbox/internal/provider/adyen"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
	"proxynth/payment-sandbox/internal/provider/fake"
	"proxynth/payment-sandbox/internal/provider/stripe"
	replaymemory "proxynth/payment-sandbox/internal/replay/adapters/memory"
	webhookhttp "proxynth/payment-sandbox/internal/webhook/adapters/http"
	webhookmemory "proxynth/payment-sandbox/internal/webhook/adapters/memory"
)

func Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return run(ctx, os.Stdout)
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

	application, err := compose(cfg, database)
	if err != nil {
		return fmt.Errorf("compose application: %w", err)
	}

	logger.Info(
		"payment sandbox starting",
		"database", cfg.Database.Path,
		"log_level", cfg.Logging.Level,
		"log_format", cfg.Logging.Format,
	)

	if err := application.serve(ctx); err != nil {
		return err
	}

	logger.Info("payment sandbox stopped")
	return nil
}

type application struct {
	server *api.Server
}

func compose(cfg config.Config, database *sql.DB) (*application, error) {
	server, err := api.NewServer(cfg.HTTP.Address)
	if err != nil {
		return nil, fmt.Errorf("create API server: %w", err)
	}

	payments := paymentsqlite.NewRepository(database)
	events := paymentsqlite.NewEventLogRepository(database)
	webhooks := webhookmemory.NewRepository()
	scenarios := replaymemory.NewRepository()
	virtualClock, err := clock.NewVirtualClock(time.Now())
	if err != nil {
		return nil, fmt.Errorf("create virtual clock: %w", err)
	}

	providers := providerdomain.NewRegistry()
	for name, provider := range map[string]providerdomain.Provider{
		"fake":   fake.New(),
		"stripe": stripe.New(),
		"adyen":  adyen.New(),
	} {
		if err := providers.Register(provider); err != nil {
			return nil, fmt.Errorf("register %s provider: %w", name, err)
		}
	}

	paymentHandler, err := paymenthttp.NewHandler(payments)
	if err != nil {
		return nil, fmt.Errorf("create payment handler: %w", err)
	}
	webhookHandler, err := webhookhttp.NewHandler(webhooks)
	if err != nil {
		return nil, fmt.Errorf("create webhook handler: %w", err)
	}
	administrationHandler, err := administrationhttp.NewHandler(virtualClock, providers)
	if err != nil {
		return nil, fmt.Errorf("create administration handler: %w", err)
	}
	scenarioHandler, err := administrationhttp.NewScenarioHandler(scenarios)
	if err != nil {
		return nil, fmt.Errorf("create scenario handler: %w", err)
	}
	timelineHandler, err := administrationhttp.NewTimelineHandler(payments, events)
	if err != nil {
		return nil, fmt.Errorf("create timeline handler: %w", err)
	}
	diagnosticsHandler, err := administrationhttp.NewDiagnosticsHandler(payments, events, virtualClock, providers)
	if err != nil {
		return nil, fmt.Errorf("create diagnostics handler: %w", err)
	}

	registrations := []struct {
		name     string
		register func() error
	}{
		{"payment", func() error { return paymentHandler.Register(server) }},
		{"webhook", func() error { return webhookHandler.Register(server) }},
		{"administration", func() error { return administrationHandler.Register(server) }},
		{"scenario", func() error { return scenarioHandler.Register(server) }},
		{"timeline", func() error { return timelineHandler.Register(server) }},
		{"diagnostics", func() error { return diagnosticsHandler.Register(server) }},
	}
	for _, registration := range registrations {
		if err := registration.register(); err != nil {
			return nil, fmt.Errorf("register %s routes: %w", registration.name, err)
		}
	}

	return &application{server: server}, nil
}

func (a *application) serve(ctx context.Context) error {
	serverErr := make(chan error, 1)
	go func() {
		err := a.server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverErr <- err
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
		a.server.SetReady(false)
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.server.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		if err := <-serverErr; err != nil {
			return fmt.Errorf("stop HTTP server: %w", err)
		}
		return nil
	}
}

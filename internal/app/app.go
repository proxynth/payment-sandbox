package app

import (
	"fmt"
	"io"
	"os"
	"proxynth/payment-sandbox/internal/platform/config"
	"proxynth/payment-sandbox/internal/platform/logging"
)

func Run() error {
	return run(os.Stdout)
}

func run(output io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	logger, err := logging.New(output, cfg.Logging)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	logger.Info("payment sandbox starting")

	return nil
}

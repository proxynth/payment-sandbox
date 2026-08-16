package config

import (
	"proxynth/payment-sandbox/internal/platform/logging"
	"time"
)

type DatabaseConfig struct {
	Path        string
	BusyTimeout time.Duration
}

type Config struct {
	Logging  logging.Config
	Database DatabaseConfig
}

func Default() Config {
	return Config{
		Logging: logging.Config{
			Level:  logging.LevelInfo,
			Format: logging.FormatText,
		},
		Database: DatabaseConfig{
			Path:        "payment-sandbox.db",
			BusyTimeout: 5 * time.Second,
		},
	}
}

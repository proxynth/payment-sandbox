package config

import "proxynth/payment-sandbox/internal/platform/logging"

type Config struct {
	Logging logging.Config
}

func Default() Config {
	return Config{
		Logging: logging.Config{
			Level:  logging.LevelInfo,
			Format: logging.FormatText,
		},
	}
}

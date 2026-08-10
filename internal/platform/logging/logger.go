package logging

import (
	"fmt"
	"io"
	"log/slog"
)

func New(w io.Writer, cfg Config) (*slog.Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	options := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler

	switch cfg.Format {
	case FormatText:
		handler = slog.NewTextHandler(w, options)
	case FormatJSON:
		handler = slog.NewJSONHandler(w, options)
	default:
		return nil, fmt.Errorf("unsupported log format: %q", cfg.Format)
	}

	return slog.New(handler), nil
}

func parseLevel(level Level) (slog.Level, error) {
	switch level {
	case LevelDebug:
		return slog.LevelDebug, nil
	case LevelInfo:
		return slog.LevelInfo, nil
	case LevelWarn:
		return slog.LevelWarn, nil
	case LevelError:
		return slog.LevelError, nil

	default:
		return 0, fmt.Errorf("invalid log level: %q", level)
	}
}

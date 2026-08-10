package logging

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

type Config struct {
	Format Format
	Level  Level
}

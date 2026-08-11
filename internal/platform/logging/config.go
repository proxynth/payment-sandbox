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

func (l Level) Valid() bool {
	switch l {
	case LevelDebug, LevelInfo, LevelWarn, LevelError:
		return true
	default:
		return false
	}
}

func (l Format) Valid() bool {
	switch l {
	case FormatText, FormatJSON:
		return true
	default:
		return false
	}
}

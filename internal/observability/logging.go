package observability

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

// NewLogger creates a new zerolog.Logger.
func NewLogger(w io.Writer, logLevel string) zerolog.Logger {
	var level zerolog.Level
	switch logLevel {
	case "debug":
		level = zerolog.DebugLevel
	case "info":
		level = zerolog.InfoLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	default:
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)
	return zerolog.New(w).With().Timestamp().Logger()
}

// NewConsoleLogger creates a new zerolog.Logger with console-friendly output.
func NewConsoleLogger(logLevel string) zerolog.Logger {
	return NewLogger(zerolog.ConsoleWriter{Out: os.Stderr}, logLevel)
}

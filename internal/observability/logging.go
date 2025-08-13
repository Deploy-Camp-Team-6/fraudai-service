package observability

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

// NewLogger creates a new zerolog.Logger.
func NewLogger(w io.Writer, debug bool) zerolog.Logger {
	logLevel := zerolog.InfoLevel
	if debug {
		logLevel = zerolog.DebugLevel
	}

	zerolog.SetGlobalLevel(logLevel)
	return zerolog.New(w).With().Timestamp().Logger()
}

// NewConsoleLogger creates a new zerolog.Logger with console-friendly output.
func NewConsoleLogger(debug bool) zerolog.Logger {
	return NewLogger(zerolog.ConsoleWriter{Out: os.Stderr}, debug)
}

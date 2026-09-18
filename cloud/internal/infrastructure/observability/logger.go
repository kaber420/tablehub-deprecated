package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// New creates a configured zerolog logger.
// In development mode, it outputs human-readable colored logs.
// In production, it outputs structured JSON.
func New(level string, isDev bool) zerolog.Logger {
	var output io.Writer

	if isDev {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	} else {
		output = os.Stdout
	}

	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.DebugLevel
	}

	return zerolog.New(output).
		Level(lvl).
		With().
		Timestamp().
		Caller().
		Str("service", "tablehub-cloud").
		Logger()
}

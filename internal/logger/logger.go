package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// New creates and returns an isolated structured logger.
// In development, it outputs human-friendly, colorized console logs.
// In production, it outputs structured JSON for log aggregation systems.
func New(env string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	var output io.Writer = os.Stdout
	if env != "production" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	return zerolog.New(output).With().Timestamp().Logger()
}

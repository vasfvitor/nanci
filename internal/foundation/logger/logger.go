package logger

import (
	"log/slog"
	"os"
)

// LevelTrace defines a custom log level for tracing.
const LevelTrace = slog.Level(-8)

// New creates a new slog.Logger configured for the CLI.
func New(verbose bool, trace bool) *slog.Logger {
	level := slog.LevelInfo
	if trace {
		level = LevelTrace
	} else if verbose {
		level = slog.LevelDebug
	}

	// We use TextHandler for CLI output
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler)
}

package logger

import (
	"log/slog"
	"os"
)

// New creates a new slog.Logger configured for the CLI.
func New(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	// We use TextHandler for CLI output
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler)
}

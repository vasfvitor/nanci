package logger

import (
	"log/slog"
	"os"
)

// LevelTrace defines a custom log level for tracing.
const LevelTrace = slog.Level(-8)

// New creates the CLI logger: text on stderr from Info, from Debug with
// verbose and from LevelTrace with trace.
func New(verbose bool, trace bool) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level(slog.LevelInfo, verbose, trace),
	}))
}

// NewJSON creates the CLI logger for --json runs: one JSON object per line on
// stderr, from Warn, so the Info progress messages stay out of the way of a
// script or an agent reading stderr. verbose and trace lower the level as in
// New.
func NewJSON(verbose bool, trace bool) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level(slog.LevelWarn, verbose, trace),
	}))
}

func level(base slog.Level, verbose, trace bool) slog.Level {
	switch {
	case trace:
		return LevelTrace
	case verbose:
		return slog.LevelDebug
	}
	return base
}

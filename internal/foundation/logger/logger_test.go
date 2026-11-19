package logger_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/vasfvitor/nanci/internal/foundation/logger"
)

func TestNewLevel(t *testing.T) {
	tests := []struct {
		name           string
		json           bool
		verbose, trace bool
		lowest         slog.Level // lowest level the logger writes
	}{
		{"default", false, false, false, slog.LevelInfo},
		{"verbose", false, true, false, slog.LevelDebug},
		{"trace", false, false, true, logger.LevelTrace},
		{"trace wins over verbose", false, true, true, logger.LevelTrace},
		{"json default", true, false, false, slog.LevelWarn},
		{"json verbose", true, true, false, slog.LevelDebug},
		{"json trace", true, false, true, logger.LevelTrace},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.New(tt.verbose, tt.trace)
			if tt.json {
				log = logger.NewJSON(tt.verbose, tt.trace)
			}
			if !log.Enabled(ctx, tt.lowest) {
				t.Errorf("level %v disabled, want enabled", tt.lowest)
			}
			if log.Enabled(ctx, tt.lowest-1) {
				t.Errorf("level %v enabled, want disabled", tt.lowest-1)
			}
		})
	}
}

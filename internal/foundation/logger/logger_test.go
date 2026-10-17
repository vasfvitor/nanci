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
		verbose, trace bool
		lowest         slog.Level // lowest level the logger writes
	}{
		{"default", false, false, slog.LevelInfo},
		{"verbose", true, false, slog.LevelDebug},
		{"trace", false, true, logger.LevelTrace},
		{"trace wins over verbose", true, true, logger.LevelTrace},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.New(tt.verbose, tt.trace)
			if !log.Enabled(ctx, tt.lowest) {
				t.Errorf("level %v disabled, want enabled", tt.lowest)
			}
			if log.Enabled(ctx, tt.lowest-1) {
				t.Errorf("level %v enabled, want disabled", tt.lowest-1)
			}
		})
	}
}

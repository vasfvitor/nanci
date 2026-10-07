package cli

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/foundation/logger"
)

// TestProdEnv_LogFlagsReachTheFactory runs the production env through the
// root and reads the log levels at RunE time, as prodAppFactory does, so the
// parsed --verbose and --trace flags must be visible there.
func TestProdEnv_LogFlagsReachTheFactory(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		envTrace  string
		wantLevel slog.Level
	}{
		{name: "no flags", wantLevel: slog.LevelInfo},
		{name: "verbose long", args: []string{"--verbose"}, wantLevel: slog.LevelDebug},
		{name: "verbose short", args: []string{"-v"}, wantLevel: slog.LevelDebug},
		{name: "trace", args: []string{"--trace"}, wantLevel: logger.LevelTrace},
		{name: "trace wins over verbose", args: []string{"-v", "--trace"}, wantLevel: logger.LevelTrace},
		{name: "NANCI_TRACE alone", envTrace: "1", wantLevel: logger.LevelTrace},
		{name: "NANCI_TRACE with verbose", args: []string{"-v"}, envTrace: "1", wantLevel: logger.LevelTrace},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NANCI_TRACE", tt.envTrace)

			// flag values stick to a command tree
			env := prodEnv()
			root := NewRootCommand(env)
			var gotLevel slog.Level
			tempCommand(t, root, "__probe", func(cmd *cobra.Command, args []string) error {
				gotLevel = enabledLevel(logger.New(logLevels(env)))
				return nil
			})
			root.SetArgs(append([]string{"__probe"}, tt.args...))
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&out)

			if err := root.ExecuteContext(context.Background()); err != nil {
				t.Fatalf("Execute returned %v", err)
			}
			if gotLevel != tt.wantLevel {
				t.Errorf("log level = %v, want %v", gotLevel, tt.wantLevel)
			}
		})
	}
}

// enabledLevel returns the lowest of the trace, debug and info levels the
// logger accepts.
func enabledLevel(log *slog.Logger) slog.Level {
	for _, level := range []slog.Level{logger.LevelTrace, slog.LevelDebug, slog.LevelInfo} {
		if log.Enabled(context.Background(), level) {
			return level
		}
	}
	return slog.LevelWarn
}

// TestProdEnv_PasswordPromptGoesToStderr pins the production streams: the
// password prompt writes to stderr so piped stdout stays clean.
func TestProdEnv_PasswordPromptGoesToStderr(t *testing.T) {
	env := prodEnv()

	if env.Out != os.Stderr {
		t.Errorf("env.Out = %s, want os.Stderr", env.Out.Name())
	}
	if env.Stdout != os.Stdout {
		t.Errorf("env.Stdout is not os.Stdout")
	}

	prompt := terminalPasswords(env)
	if prompt.Out != os.Stderr {
		t.Errorf("password prompt writes to %s, want os.Stderr", prompt.Out.Name())
	}
	if prompt.In != os.Stdin {
		t.Errorf("password prompt reads from %s, want os.Stdin", prompt.In.Name())
	}
}

package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// resetRoot restores the package-global rootCmd to a clean state between tests.
// Cobra accumulates args and IO overrides on a command instance; tests must clear them.
// Persistent flags are registered once at package init and must not be reset.
func resetRoot(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})
}

// tempCommand registers a throwaway subcommand on rootCmd for the duration of one
// test and removes it via t.Cleanup so it never leaks into other tests.
func tempCommand(t *testing.T, name string, runE func(cmd *cobra.Command, args []string) error) *cobra.Command {
	t.Helper()
	sub := &cobra.Command{
		Use:           name,
		Short:         "test-only command",
		RunE:          runE,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.AddCommand(sub)
	t.Cleanup(func() {
		rootCmd.RemoveCommand(sub)
	})
	return sub
}

// TestExecute_RuntimeErrorDoesNotPrintUsage asserts that when a subcommand's RunE
// returns an error, Cobra stays silent (no usage text, no error echo) because the
// root has SilenceUsage and SilenceErrors. The caller (main.go) is the sole
// reporting boundary and is expected to print the returned error to stderr.
func TestExecute_RuntimeErrorDoesNotPrintUsage(t *testing.T) {
	resetRoot(t)
	wantErr := errors.New("boom")
	tempCommand(t, "__test_error", func(cmd *cobra.Command, args []string) error {
		return wantErr
	})

	rootCmd.SetArgs([]string{"__test_error"})
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)

	gotErr := Execute(context.Background())

	if !errors.Is(gotErr, wantErr) {
		t.Fatalf("Execute returned %v, want %v", gotErr, wantErr)
	}
	if out.Len() != 0 {
		t.Errorf("stdout not empty: %q", out.String())
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr not empty (expected SilenceUsage + SilenceErrors): %q", errOut.String())
	}
}

// TestExecute_UnknownCommandReturnsError asserts that an unknown subcommand yields
// an error from Execute without Cobra printing usage to stderr.
func TestExecute_UnknownCommandReturnsError(t *testing.T) {
	resetRoot(t)
	rootCmd.SetArgs([]string{"__definitely_not_a_real_command"})
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)

	gotErr := Execute(context.Background())

	if gotErr == nil {
		t.Fatal("Execute returned nil for unknown command, want error")
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr not empty (expected silence): %q", errOut.String())
	}
}

// TestRootPersistentFlagsParse asserts that the persistent --verbose and --trace
// flags still parse and flip their package-global toggles when Execute runs.
func TestRootPersistentFlagsParse(t *testing.T) {
	resetRoot(t)
	prevVerbose, prevTrace := verbose, trace
	t.Cleanup(func() {
		verbose, trace = prevVerbose, prevTrace
	})
	verbose, trace = false, false

	// No subcommand: Cobra runs the root's help, returning nil. The persistent
	// flags are still parsed before dispatch.
	rootCmd.SetArgs([]string{"--verbose", "--trace"})
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)

	if err := Execute(context.Background()); err != nil {
		t.Fatalf("Execute returned %v, want nil", err)
	}
	if !verbose {
		t.Errorf("verbose flag not applied after Execute")
	}
	if !trace {
		t.Errorf("trace flag not applied after Execute")
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Errorf("expected help output on stdout when no subcommand given")
	}
}

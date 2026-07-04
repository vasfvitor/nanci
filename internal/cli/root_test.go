package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/vasfvitor/nanci/internal/app"
)

// tempCommand registers a throwaway subcommand on the given root for the
// duration of one test and removes it via t.Cleanup so it never leaks.
func tempCommand(t *testing.T, root *cobra.Command, name string, runE func(cmd *cobra.Command, args []string) error) *cobra.Command {
	t.Helper()
	sub := &cobra.Command{
		Use:           name,
		Short:         "test-only command",
		RunE:          runE,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(sub)
	t.Cleanup(func() {
		root.RemoveCommand(sub)
	})
	return sub
}

// newTestRoot builds a fresh root with stub Verbose/Trace booleans and a
// no-op AppFactory. Tests pass their own subcommands via tempCommand.
func newTestRoot(t *testing.T) (*cobra.Command, *bool, *bool) {
	t.Helper()
	v, tr := false, false
	root := NewRootCommand(CommandEnv{
		AppFactory: func(context.Context) (*app.App, func(), error) {
			return nil, func() {}, nil
		},
		Verbose: &v,
		Trace:   &tr,
	})
	return root, &v, &tr
}

// TestExecute_RuntimeErrorDoesNotPrintUsage asserts that when a subcommand's RunE
// returns an error, Cobra stays silent (no usage text, no error echo) because the
// root has SilenceUsage and SilenceErrors. The caller (main.go) is the sole
// reporting boundary and is expected to print the returned error to stderr.
func TestExecute_RuntimeErrorDoesNotPrintUsage(t *testing.T) {
	root, _, _ := newTestRoot(t)
	wantErr := errors.New("boom")
	tempCommand(t, root, "__test_error", func(cmd *cobra.Command, args []string) error {
		return wantErr
	})

	root.SetArgs([]string{"__test_error"})
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)

	gotErr := root.ExecuteContext(context.Background())

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
	root, _, _ := newTestRoot(t)
	root.SetArgs([]string{"__definitely_not_a_real_command"})
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)

	gotErr := root.ExecuteContext(context.Background())

	if gotErr == nil {
		t.Fatal("Execute returned nil for unknown command, want error")
	}
	if errOut.Len() != 0 {
		t.Errorf("stderr not empty (expected silence): %q", errOut.String())
	}
}

// TestRootPersistentFlagsParse asserts that the persistent --verbose and --trace
// flags still parse and flip the env.Verbose/env.Trace pointers when the
// command runs.
func TestRootPersistentFlagsParse(t *testing.T) {
	root, v, tr := newTestRoot(t)

	// No subcommand: the root's RunE returns cmd.Help(), so Execute returns nil.
	root.SetArgs([]string{"--verbose", "--trace"})
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)

	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("Execute returned %v, want nil", err)
	}
	if !*v {
		t.Errorf("verbose flag not applied after Execute")
	}
	if !*tr {
		t.Errorf("trace flag not applied after Execute")
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Errorf("expected help output on stdout when no subcommand given")
	}
}

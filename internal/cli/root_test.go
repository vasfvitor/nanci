package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
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
// Stderr is a temp file: NewRootCommand hands it to cobra, and a nil
// *os.File would drop what is written to it.
func newTestRoot(t *testing.T) (*cobra.Command, *bool, *bool) {
	t.Helper()
	stderr, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stderr.Close() })
	v, tr := false, false
	root := NewRootCommand(CommandEnv{
		Stderr: stderr,
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

// TestRun_UsageErrorsAsJSON checks that a command line cobra refuses reaches
// stderr as an invalid_input envelope, also when --json comes after the bad
// flag, where the flag parser never reads it.
func TestRun_UsageErrorsAsJSON(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantJSON bool
		wantCode string
	}{
		{"json after an unknown flag", []string{"nfe", "list", "-c", nfeTestCNPJ, "--bogus", "--json"}, true, app.ErrorCodeInvalidInput},
		{"json before an unknown flag", []string{"--json", "nfe", "list", "-c", nfeTestCNPJ, "--bogus"}, true, app.ErrorCodeInvalidInput},
		{"missing required flag", []string{"nfe", "manifestar", "-c", nfeTestCNPJ, "--json"}, true, app.ErrorCodeInvalidInput},
		{"wrong number of arguments", []string{"nfe", "import", "-c", nfeTestCNPJ, "--json"}, true, app.ErrorCodeInvalidInput},
		{"json=false", []string{"nfe", "list", "--bogus", "--json=false"}, false, ""},
		{"no json", []string{"nfe", "list", "--bogus"}, false, ""},
		// The command ran and failed: not a usage error.
		{"error from the command", []string{"nfe", "list", "-c", nfeTestCNPJ, "--json"}, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			stderr, err := os.CreateTemp(t.TempDir(), "stderr")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = stderr.Close() }()
			v, tr, js := false, false, false
			code := run(context.Background(), CommandEnv{
				Stdout: &stdout,
				Stderr: stderr,
				AppFactory: func(context.Context) (*app.App, func(), error) {
					return nil, nil, errors.New("sem banco no teste")
				},
				Verbose: &v,
				Trace:   &tr,
				JSON:    &js,
			}, tt.args)
			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want nothing", stdout.String())
			}
			written, err := os.ReadFile(stderr.Name())
			if err != nil {
				t.Fatal(err)
			}
			var envelope errorEnvelope
			isJSON := json.Unmarshal(written, &envelope) == nil
			if isJSON != tt.wantJSON {
				t.Fatalf("stderr as JSON = %t, want %t: %q", isJSON, tt.wantJSON, written)
			}
			if tt.wantJSON && (envelope.Error.Code != tt.wantCode || envelope.Error.Message == "") {
				t.Errorf("envelope = %+v, want code %q and a message", envelope, tt.wantCode)
			}
		})
	}
}

func TestJSONRequested(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"nfe", "list"}, false},
		{[]string{"nfe", "list", "--json"}, true},
		{[]string{"--json=true", "nfe"}, true},
		{[]string{"--json", "--json=false"}, false},
		{[]string{"nfe", "import", "--", "--json"}, false},
		{[]string{"--jsonx"}, false},
	}
	for _, tt := range tests {
		if got := jsonRequested(tt.args); got != tt.want {
			t.Errorf("jsonRequested(%q) = %t, want %t", tt.args, got, tt.want)
		}
	}
}

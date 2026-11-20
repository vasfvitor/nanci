package cli

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// Execute runs the root command and returns the process exit code. It is the
// single reporting boundary: it writes a failed command's error to stderr, as
// text or, with --json, as a JSON envelope, and returns 1. Cobra itself stays
// silent (SilenceUsage + SilenceErrors).
//
// Execute builds a fresh *cobra.Command tree from prodEnv() on every call so
// that tests can re-invoke Execute against fakes without sharing cobra state.
func Execute(ctx context.Context) int {
	return run(ctx, prodEnv(), os.Args[1:])
}

// run is Execute with the env and the arguments given, for tests.
func run(ctx context.Context, env CommandEnv, args []string) int {
	root := NewRootCommand(env) //nolint:contextcheck // cobra hands ctx to every RunE through cmd.Context().
	root.SetArgs(args)
	// Cobra checks the flags, the arguments and the command name before it
	// calls RunE, so an error without RunE having started is a usage error.
	started := false
	markStart(root, &started)

	err := root.ExecuteContext(ctx)
	if err == nil {
		return 0
	}
	asJSON := env.jsonOutput()
	if !started {
		err = usageError{err}
		// The flag parser stops at the first bad flag, so a --json after it
		// was never read.
		asJSON = asJSON || jsonRequested(args)
	}
	reportError(env.Stderr, err, asJSON)
	return 1
}

// markStart wraps the RunE of cmd and of every command under it to set
// *started when it is called.
func markStart(cmd *cobra.Command, started *bool) {
	if runE := cmd.RunE; runE != nil {
		cmd.RunE = func(c *cobra.Command, args []string) error {
			*started = true
			return runE(c, args)
		}
	}
	for _, sub := range cmd.Commands() {
		markStart(sub, started)
	}
}

// usageError is a command line that cobra refused before running the
// command: an unknown flag or command, a missing required flag, the wrong
// number of arguments. --json reports it as invalid_input.
type usageError struct{ err error }

func (e usageError) Error() string { return e.err.Error() }
func (e usageError) Unwrap() error { return e.err }

// jsonRequested reports whether args turn --json on, read without the flag
// parser. The last occurrence wins, as in the parser; "--" ends the flags.
func jsonRequested(args []string) bool {
	on := false
	for _, arg := range args {
		if arg == "--" {
			break
		}
		if arg == "--json" {
			on = true
		} else if value, ok := strings.CutPrefix(arg, "--json="); ok {
			on, _ = strconv.ParseBool(value)
		}
	}
	return on
}

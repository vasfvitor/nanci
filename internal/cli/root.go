package cli

import (
	"context"
)

// Execute runs the root command and returns the process exit code. It is the
// single reporting boundary: it writes a failed command's error to stderr, as
// text or, with --json, as a JSON envelope, and returns 1. Cobra itself stays
// silent (SilenceUsage + SilenceErrors).
//
// Execute builds a fresh *cobra.Command tree from prodEnv() on every call so
// that tests can re-invoke Execute against fakes without sharing cobra state.
func Execute(ctx context.Context) int {
	env := prodEnv()
	err := NewRootCommand(env).ExecuteContext(ctx) //nolint:contextcheck // cobra hands ctx to every RunE through cmd.Context().
	if err == nil {
		return 0
	}
	reportError(env.Stderr, err, env.jsonOutput())
	return 1
}

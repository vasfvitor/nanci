package cli

import (
	"context"
)

// Execute runs the root command and returns any error from the command tree.
// The caller is the single reporting boundary: it prints the error to stderr
// and chooses the exit code; Cobra itself stays silent (SilenceUsage + SilenceErrors).
//
// Execute builds a fresh *cobra.Command tree from prodEnv() on every call so
// that tests can re-invoke Execute against fakes without sharing cobra state.
func Execute(ctx context.Context) error {
	return NewRootCommand(prodEnv()).ExecuteContext(ctx)
}

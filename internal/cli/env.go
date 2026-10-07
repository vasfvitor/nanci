package cli

import (
	"context"
	"io"
	"os"

	"github.com/vasfvitor/nanci/internal/app"
)

// CommandEnv is the dependency-injection seam for the CLI command tree.
// Every subcommand and the root command are built from a CommandEnv; the
// package no longer relies on package-global state for runtime wiring.
//
// In/Out are *os.File because TerminalCredentialProvider passes them to
// golang.org/x/term, which requires an Fd(). Stdout is an io.Writer so
// the fresh root's cmd.SetOut can route through anything (e.g. a test
// bytes.Buffer).
type CommandEnv struct {
	In         *os.File
	Out        *os.File // stderr, for the password prompt
	Stdout     io.Writer
	AppFactory AppFactory
	Verbose    *bool
	Trace      *bool
}

// AppFactory constructs an *app.App on demand. Returning the same instance
// from a captured var is acceptable; constructing per-call is the norm.
// cleanup() must be safe to call multiple times and is invoked by every
// RunE via defer, regardless of how many times AppFactory is called.
type AppFactory func(ctx context.Context) (*app.App, func(), error)

// prodEnv builds a CommandEnv wired to the process's real IO and the
// production factory. The factory takes the env itself, so the streams
// reach it by field name and cannot be passed in the wrong order.
func prodEnv() CommandEnv {
	v, tr := false, false
	env := CommandEnv{
		In:      os.Stdin,
		Out:     os.Stderr,
		Stdout:  os.Stdout,
		Verbose: &v,
		Trace:   &tr,
	}
	env.AppFactory = prodAppFactory(env)
	return env
}

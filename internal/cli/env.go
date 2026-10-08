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
// Stdin and Stderr are *os.File because TerminalCredentialProvider passes them
// to golang.org/x/term, which requires an Fd().
type CommandEnv struct {
	Stdin      *os.File
	Stderr     *os.File
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
		Stdin:   os.Stdin,
		Stderr:  os.Stderr,
		Stdout:  os.Stdout,
		Verbose: &v,
		Trace:   &tr,
	}
	env.AppFactory = prodAppFactory(env)
	return env
}

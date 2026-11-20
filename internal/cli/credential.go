package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
)

// errNoPasswordTerminal is the failure when the certificate password is
// neither saved nor in NANCI_CERT_PASSWORD and stdin is not a terminal.
var errNoPasswordTerminal = fmt.Errorf("%w: execute uma vez num terminal para salvá-la, ou defina NANCI_CERT_PASSWORD", app.ErrPasswordRequired)

// TerminalCredentialProvider obtains certificate passwords from the terminal.
// It first checks the NANCI_CERT_PASSWORD environment variable, then prompts
// the user interactively. This is the CLI adapter for app.CredentialProvider.
type TerminalCredentialProvider struct {
	In  *os.File // typically os.Stdin
	Out *os.File // typically os.Stderr (keeps stdout clean for piping)
}

// GetCertPassword implements app.CredentialProvider.
//
// The password is returned as a []byte so the caller can zero it after use.
func (p TerminalCredentialProvider) GetCertPassword(ctx context.Context, req app.CertPasswordRequest) ([]byte, error) {
	// 1. Try environment variable first (non-interactive / CI use)
	// os.Getenv returns a string, which cannot be zeroed; the environment keeps
	// its own copy for the whole process anyway.
	if pass := os.Getenv("NANCI_CERT_PASSWORD"); pass != "" {
		return []byte(pass), nil
	}

	// 2. Prompt interactively, when there is a terminal to read from. A
	// script or an AI agent has none: fail with what the person can do
	// instead of writing a prompt nobody answers.
	if !term.IsTerminal(int(p.In.Fd())) {
		return nil, errNoPasswordTerminal
	}
	if _, err := fmt.Fprintf(p.Out, "Digite a senha do certificado '%s' (%s) para consultar %s: ", req.CredentialLabel, req.CertPath, req.TargetCNPJ); err != nil {
		return nil, fmt.Errorf("falha ao exibir prompt da senha: %w", err)
	}
	// term.ReadPassword already returns []byte; pass it straight through so the
	// password never becomes an unzeroable string.
	fd := int(p.In.Fd())
	state, stateErr := term.GetState(fd)
	restore := func() {
		if stateErr == nil {
			_ = term.Restore(fd, state)
		}
	}
	bytePassword, err := readPasswordContext(ctx, func() ([]byte, error) { return term.ReadPassword(fd) }, restore)
	if _, newlineErr := fmt.Fprintln(p.Out); newlineErr != nil && err == nil {
		cert.ZeroBytes(bytePassword)
		return nil, fmt.Errorf("falha ao finalizar prompt da senha: %w", newlineErr)
	}
	if errors.Is(err, app.ErrOperationCanceled) {
		return nil, err
	}
	if err != nil {
		cert.ZeroBytes(bytePassword)
		return nil, fmt.Errorf("falha ao ler a senha: %w", err)
	}

	return bytePassword, nil
}

// readPasswordContext runs read until it returns or ctx ends. term.ReadPassword
// does not watch ctx, so without this Ctrl+C at the prompt (which cancels ctx
// in cmd/nanci) would wait for Enter. On cancel it calls restore, to turn the
// echo back on, and returns app.ErrOperationCanceled; the read is left
// blocked on stdin until the process exits.
func readPasswordContext(ctx context.Context, read func() ([]byte, error), restore func()) ([]byte, error) {
	type result struct {
		pass []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		pass, err := read()
		done <- result{pass, err}
	}()
	select {
	case r := <-done:
		return r.pass, r.err
	case <-ctx.Done():
		restore()
		return nil, app.ErrOperationCanceled
	}
}

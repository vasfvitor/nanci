package cli

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/term"
	"github.com/vasfvitor/nanci/internal/app"
)

// TerminalCredentialProvider obtains certificate passwords from the terminal.
// It first checks the NANCI_CERT_PASSWORD environment variable, then prompts
// the user interactively. This is the CLI adapter for app.CredentialProvider.
type TerminalCredentialProvider struct {
	In  *os.File // typically os.Stdin
	Out *os.File // typically os.Stderr (keeps stdout clean for piping)
}

// GetCertPassword implements app.CredentialProvider.
func (p TerminalCredentialProvider) GetCertPassword(ctx context.Context, req app.CertPasswordRequest) (string, error) {
	// 1. Try environment variable first (non-interactive / CI use)
	if pass := os.Getenv("NANCI_CERT_PASSWORD"); pass != "" {
		return pass, nil
	}

	// 2. Prompt interactively
	fmt.Fprintf(p.Out, "Digite a senha do certificado para %s: ", req.CertPath)
	bytePassword, err := term.ReadPassword(int(p.In.Fd()))
	fmt.Fprintln(p.Out) // newline after the silent input
	if err != nil {
		return "", fmt.Errorf("falha ao ler a senha: %w", err)
	}

	return string(bytePassword), nil
}

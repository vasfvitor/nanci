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
func (p TerminalCredentialProvider) GetCertPassword(ctx context.Context, req app.CertPasswordRequest) ([]byte, error) {
	// 1. Try environment variable first (non-interactive / CI use)
	if pass := os.Getenv("NANCI_CERT_PASSWORD"); pass != "" {
		return []byte(pass), nil
	}

	// 2. Prompt interactively
	if _, err := fmt.Fprintf(p.Out, "Digite a senha do certificado '%s' (%s) para consultar %s: ", req.CredentialLabel, req.CertPath, req.TargetCNPJ); err != nil {
		return nil, fmt.Errorf("falha ao exibir prompt da senha: %w", err)
	}
	bytePassword, err := term.ReadPassword(int(p.In.Fd()))
	if _, newlineErr := fmt.Fprintln(p.Out); newlineErr != nil && err == nil {
		return nil, fmt.Errorf("falha ao finalizar prompt da senha: %w", newlineErr)
	}
	if err != nil {
		return nil, fmt.Errorf("falha ao ler a senha: %w", err)
	}

	return bytePassword, nil
}

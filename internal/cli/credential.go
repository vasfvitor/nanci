package cli

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/foundation/cert"
)

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

	// 2. Prompt interactively
	if _, err := fmt.Fprintf(p.Out, "Digite a senha do certificado '%s' (%s) para consultar %s: ", req.CredentialLabel, req.CertPath, req.TargetCNPJ); err != nil {
		return nil, fmt.Errorf("falha ao exibir prompt da senha: %w", err)
	}
	// term.ReadPassword already returns []byte; pass it straight through so the
	// password never becomes an unzeroable string.
	bytePassword, err := term.ReadPassword(int(p.In.Fd()))
	if _, newlineErr := fmt.Fprintln(p.Out); newlineErr != nil && err == nil {
		cert.ZeroBytes(bytePassword)
		return nil, fmt.Errorf("falha ao finalizar prompt da senha: %w", newlineErr)
	}
	if err != nil {
		cert.ZeroBytes(bytePassword)
		return nil, fmt.Errorf("falha ao ler a senha: %w", err)
	}

	return bytePassword, nil
}

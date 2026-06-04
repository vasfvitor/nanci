package commands

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/term"
)

// CLICredentialProvider implements nfse.CredentialProvider for the command line.
// It tries to read from an environment variable first.
// If not found, it prompts the user securely in the terminal.
type CLICredentialProvider struct{}

func (p *CLICredentialProvider) GetCertPassword(ctx context.Context, certPath string) ([]byte, error) {
	// 1. Try to read from environment variable
	if pass := os.Getenv("NANCI_CERT_PASSWORD"); pass != "" {
		return []byte(pass), nil
	}

	// 2. Prompt interactively
	fmt.Printf("Digite a senha do certificado para %s: ", certPath)
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("falha ao ler a senha: %w", err)
	}
	fmt.Println() // Add a new line after reading password

	return bytePassword, nil
}

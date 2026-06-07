package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"

	"github.com/vasfvitor/nanci/internal/foundation/cert"
)

const keyringService = "nanci_certs"

// KeyringCredentialProvider wraps an existing CredentialProvider.
// It attempts to retrieve the certificate password from the OS native credential manager (keyring).
// If it fails or the stored password is invalid, it falls back to the underlying provider
// and, upon success, saves the new password to the keyring.
type KeyringCredentialProvider struct {
	Fallback CredentialProvider
}

// GetCertPassword implements CredentialProvider.
func (p KeyringCredentialProvider) GetCertPassword(ctx context.Context, req CertPasswordRequest) (string, error) {
	// 1. Try to get the password from the OS keyring
	pass, err := keyring.Get(keyringService, req.CredentialID)
	if err == nil && pass != "" {
		// Verify if the retrieved password actually works for this certificate
		if _, err := cert.LoadPKCS12(req.CertPath, pass); err == nil {
			// Password is valid
			return pass, nil
		}
		// If it's invalid, we ignore the error and fall back to the user prompt
	}

	// 2. Fall back to the wrapped provider (CLI or Wails)
	pass, err = p.Fallback.GetCertPassword(ctx, req)
	if err != nil {
		return "", err
	}

	// 3. Verify the user-provided password before saving it
	if _, err := cert.LoadPKCS12(req.CertPath, pass); err != nil {
		if errors.Is(err, cert.ErrInvalidPass) {
			return "", fmt.Errorf("senha inválida para o certificado")
		}
		return "", fmt.Errorf("erro ao verificar senha: %w", err)
	}

	// 4. Save the valid password to the keyring for future use
	_ = keyring.Set(keyringService, req.CredentialID, pass) // We ignore the error as it's not fatal

	return pass, nil
}

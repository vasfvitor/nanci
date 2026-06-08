package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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
	Log      *slog.Logger
}

// GetCertPassword implements CredentialProvider.
func (p KeyringCredentialProvider) GetCertPassword(ctx context.Context, req CertPasswordRequest) ([]byte, error) {
	// 1. Try to get the password from the OS keyring
	passStr, err := keyring.Get(keyringService, req.CredentialID)
	if err == nil && passStr != "" {
		passBytes := []byte(passStr)
		// Verify if the retrieved password actually works for this certificate
		if _, err := cert.LoadPKCS12(req.CertPath, passBytes); err == nil {
			// Password is valid
			return passBytes, nil
		}
		// If it's invalid, we ignore the error and fall back to the user prompt
		cert.ZeroBytes(passBytes)
	}

	// 2. Fall back to the wrapped provider (CLI or Wails)
	passBytes, err := p.Fallback.GetCertPassword(ctx, req)
	if err != nil {
		return nil, err
	}

	// 3. Verify the user-provided password before saving it
	if _, err := cert.LoadPKCS12(req.CertPath, passBytes); err != nil {
		cert.ZeroBytes(passBytes)
		if errors.Is(err, cert.ErrInvalidPass) {
			return nil, fmt.Errorf("senha inválida para o certificado")
		}
		return nil, fmt.Errorf("erro ao verificar senha: %w", err)
	}

	// 4. Save the valid password to the keyring for future use
	errSet := keyring.Set(keyringService, req.CredentialID, string(passBytes))
	if errSet != nil && p.Log != nil {
		p.Log.WarnContext(ctx, "Falha ao gravar senha no chaveiro nativo do S.O.",
			slog.String("credential_id", req.CredentialID),
			slog.String("cert_path", req.CertPath),
			slog.String("error", errSet.Error()))
	}

	return passBytes, nil
}

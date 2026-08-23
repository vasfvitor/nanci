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

// keyringGet and keyringSet wrap the OS keyring so tests can stub them.
var (
	keyringGet = keyring.Get
	keyringSet = keyring.Set
)

// KeyringCredentialProvider wraps an existing CredentialProvider.
// It attempts to retrieve the certificate password from the OS native credential manager (keyring).
// If it fails or the stored password is invalid, it falls back to the underlying provider
// and, upon success, saves the new password to the keyring.
type KeyringCredentialProvider struct {
	Fallback CredentialProvider
	// Log receives keyring diagnostics. Nil disables logging.
	Log *slog.Logger
}

// GetCertPassword implements CredentialProvider.
//
// The OS keyring API is string-based, so the password crosses that boundary as
// an unzeroable string. Everywhere else it is a []byte, and the caller owns the
// returned slice: it should zero it with cert.ZeroBytes once done.
func (p KeyringCredentialProvider) GetCertPassword(ctx context.Context, req CertPasswordRequest) ([]byte, error) {
	log := p.logger()

	// 1. Try to get the password from the OS keyring
	stored, err := keyringGet(keyringService, req.CredentialID)
	switch {
	case err == nil && stored != "":
		// Verify if the retrieved password actually works for this certificate
		storedPass := []byte(stored)
		if _, err := cert.LoadPKCS12(req.CertPath, storedPass); err == nil {
			return storedPass, nil
		}
		cert.ZeroBytes(storedPass)
		log.DebugContext(ctx, "senha do keyring não abre o certificado; solicitando nova senha",
			slog.String("credential_id", req.CredentialID),
			slog.String("cert_path", req.CertPath))
	case err != nil && !errors.Is(err, keyring.ErrNotFound):
		log.DebugContext(ctx, "falha ao ler senha do keyring; solicitando senha",
			slog.String("credential_id", req.CredentialID),
			slog.String("error", err.Error()))
	}

	// 2. Fall back to the wrapped provider (CLI or Wails)
	pass, err := p.Fallback.GetCertPassword(ctx, req)
	if err != nil {
		return nil, err
	}

	// 3. Verify the user-provided password before saving it
	if _, err := cert.LoadPKCS12(req.CertPath, pass); err != nil {
		cert.ZeroBytes(pass)
		if errors.Is(err, cert.ErrInvalidPass) {
			return nil, fmt.Errorf("senha inválida para o certificado")
		}
		return nil, fmt.Errorf("erro ao verificar senha: %w", err)
	}

	// 4. Save the valid password to the keyring for future use. Failure is not
	// fatal, but it means the user will be prompted again on every run, so
	// make it visible.
	if err := keyringSet(keyringService, req.CredentialID, string(pass)); err != nil {
		log.WarnContext(ctx, "falha ao salvar senha no keyring; a senha será solicitada novamente",
			slog.String("credential_id", req.CredentialID),
			slog.String("cert_path", req.CertPath),
			slog.String("error", err.Error()))
	}

	return pass, nil
}

func (p KeyringCredentialProvider) logger() *slog.Logger {
	if p.Log != nil {
		return p.Log
	}
	return slog.New(slog.DiscardHandler)
}

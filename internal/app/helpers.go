package app

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
)

func normalizeCNPJ(raw string) (string, error) {
	if err := cnpj.Validate(raw); err != nil {
		return "", fmt.Errorf("CNPJ inválido: %w", err)
	}
	return cnpj.Clean(raw), nil
}

func (a *App) companyByCNPJ(ctx context.Context, raw string) (*nfse.Company, error) {
	cleanedCNPJ, err := normalizeCNPJ(raw)
	if err != nil {
		return nil, err
	}

	company, err := a.CompanyRepo.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("empresa não encontrada para o CNPJ %s", cnpj.Format(cleanedCNPJ))
		}
		return nil, fmt.Errorf("buscar empresa: %w", err)
	}
	return company, nil
}

func (a *App) credentialByID(ctx context.Context, id nfse.CredentialID) (*nfse.Credential, error) {
	credential, err := a.CredentialRepo.CredentialByID(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("credencial não encontrada")
		}
		return nil, fmt.Errorf("buscar credencial: %w", err)
	}
	return credential, nil
}

func validateCertificatePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("arquivo de certificado não encontrado: %s", path)
		}
		return fmt.Errorf("verificar certificado: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("caminho do certificado aponta para um diretório: %s", path)
	}
	return nil
}

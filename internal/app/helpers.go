package app

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/foundation/cnpj"
	"github.com/vasfvitor/nanci/internal/nfse"
)

func normalizeCNPJ(raw string) (string, error) {
	if err := cnpj.Validate(raw); err != nil {
		return "", fmt.Errorf("CNPJ inválido: %w", err)
	}
	return cnpj.Clean(raw), nil
}

func lookupCompanyByCNPJ(ctx context.Context, repo *company.Store, raw string) (*nfse.Company, error) {
	cleanedCNPJ, err := normalizeCNPJ(raw)
	if err != nil {
		return nil, err
	}

	comp, err := repo.CompanyByCNPJ(ctx, cleanedCNPJ)
	if err != nil {
		if errors.Is(err, company.ErrCompanyNotFound) {
			return nil, fmt.Errorf("empresa não encontrada para o CNPJ %s", cnpj.Format(cleanedCNPJ))
		}
		return nil, fmt.Errorf("buscar empresa: %w", err)
	}
	return comp, nil
}

func lookupCredentialByID(ctx context.Context, repo *credential.Store, id nfse.CredentialID) (*nfse.Credential, error) {
	cred, err := repo.CredentialByID(ctx, id)
	if err != nil {
		if errors.Is(err, credential.ErrCredentialNotFound) {
			return nil, fmt.Errorf("credencial não encontrada")
		}
		return nil, fmt.Errorf("buscar credencial: %w", err)
	}
	return cred, nil
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

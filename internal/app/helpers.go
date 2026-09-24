package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// tempPathFor names the file an export is written to before it replaces
// outPath: "out.zip" becomes "out.tmp.zip".
func tempPathFor(outPath string) string {
	ext := filepath.Ext(outPath)
	return strings.TrimSuffix(outPath, ext) + ".tmp" + ext
}

// writeViaTemp lets gen write the export to a temp file and then moves it to
// outPath, so a failed export never leaves a partial file at outPath.
func writeViaTemp(outPath string, gen func(tempPath string) error) error {
	tempPath := tempPathFor(outPath)
	defer func() { _ = os.Remove(tempPath) }()
	if err := gen(tempPath); err != nil {
		return fmt.Errorf("gerar arquivo: %w", err)
	}
	if err := os.Rename(tempPath, outPath); err != nil {
		return fmt.Errorf("mover arquivo temporário para destino final: %w", err)
	}
	return nil
}

// writeFileAtomic writes data to path through a temp file. what names the
// content in error messages ("XML", "DANFSe").
func writeFileAtomic(path string, data []byte, what string) error {
	tempPath := tempPathFor(path)
	defer func() { _ = os.Remove(tempPath) }()
	if err := os.WriteFile(tempPath, data, 0o644); err != nil { // #nosec G306 -- exported files are meant to be shared.
		return fmt.Errorf("gravar %s temp: %w", what, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("mover %s temp: %w", what, err)
	}
	return nil
}

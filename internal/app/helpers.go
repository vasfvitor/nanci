package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// parseKeys validates and normalizes the chaves a user typed with parse. An
// invalid one fails the whole call.
func parseKeys(raw []string, parse func(string) (string, error)) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	chaves := make([]string, 0, len(raw))
	for _, r := range raw {
		key, err := parse(r)
		if err != nil {
			return nil, fmt.Errorf("chave de acesso inválida %q: %w", strings.TrimSpace(r), err)
		}
		chaves = append(chaves, key)
	}
	return chaves, nil
}

// parseAccessKeys parses 44-digit NF-e/CT-e chaves. An invalid one fails
// with an error matching dfe.ErrInvalidAccessKey.
func parseAccessKeys(raw []string) ([]string, error) {
	return parseKeys(raw, func(r string) (string, error) {
		key, err := dfe.ParseAccessKey(r)
		return string(key), err
	})
}

// parseNFSeAccessKeys parses 50-digit NFS-e chaves. The "NFS" + 50-digit
// form of the infNFSe Id is accepted too and returned as given, because
// migration 020 leaves a prefixed row whose digits another document already
// has, and a chave must match the row as stored.
func parseNFSeAccessKeys(raw []string) ([]string, error) {
	return parseKeys(raw, func(r string) (string, error) {
		trimmed := strings.TrimSpace(r)
		if _, err := nfse.ParseInfNFSeID(trimmed); err != nil {
			return "", err
		}
		return trimmed, nil
	})
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

package report

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// GenerateZIP creates a ZIP archive containing the physical XML files for the given company-facing documents.
// baseDir is the root data directory where "xml/" is located.
func GenerateZIP(documents []nfse.CompanyDocument, baseDir string, outPath string) (err error) {
	zipFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer func() {
		if cerr := zipFile.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close zip file: %w", cerr)
		}
	}()

	zipWriter := zip.NewWriter(zipFile)
	defer func() {
		if cerr := zipWriter.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close zip writer: %w", cerr)
		}
	}()

	for _, doc := range documents {
		if doc.XMLPath == "" {
			continue // Skip if no physical file was registered
		}

		fullPath := filepath.Join(baseDir, doc.XMLPath)
		fileToZip, err := os.Open(fullPath)
		if err != nil {
			// Instead of failing the entire process, we log/print or skip
			fmt.Printf("[Aviso] Arquivo XML não encontrado para chave %s: %v\n", doc.ChaveAcesso, err)
			continue
		}

		// The path inside the zip file
		roleFolder := doc.CompanyRole
		if roleFolder == "" || roleFolder == "none" {
			roleFolder = "sem-papel-fiscal"
		}
		zipEntryPath := fmt.Sprintf("%s/%s/%s.xml", doc.Competence, roleFolder, doc.ChaveAcesso)

		writer, err := zipWriter.Create(zipEntryPath)
		if err != nil {
			_ = fileToZip.Close()
			return fmt.Errorf("failed to create zip entry for %s: %w", doc.ChaveAcesso, err)
		}

		if _, err := io.Copy(writer, fileToZip); err != nil {
			_ = fileToZip.Close()
			return fmt.Errorf("failed to write file %s to zip: %w", doc.ChaveAcesso, err)
		}

		_ = fileToZip.Close()
	}

	return nil
}

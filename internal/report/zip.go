package report

import (
	"archive/zip"
	"fmt"
	"os"

	"github.com/vasfvitor/nanci/internal/files"
)

// GenerateZIP creates a ZIP archive containing the physical XML files for the given documents.
func GenerateZIP(documents []ReportRow, xmlStore files.XMLStore, outPath string) (err error) {
	zipFile, err := os.Create(outPath) // #nosec G304 -- destination is explicitly selected by the local user.
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
		if doc.RawHash == "" {
			continue // Skip if no physical file was registered
		}

		data, err := xmlStore.Get(doc.RawHash)
		if err != nil {
			fmt.Printf("[Aviso] Arquivo XML não encontrado para chave %s: %v\n", doc.ChaveAcesso, err)
			continue
		}

		// The path inside the zip file
		roleFolder := string(doc.CompanyRole)
		if roleFolder == "" || roleFolder == "none" {
			roleFolder = "sem-papel-fiscal"
		}
		zipEntryPath := fmt.Sprintf("%s/%s/%s.xml", doc.Competence, roleFolder, doc.ChaveAcesso)

		writer, err := zipWriter.Create(zipEntryPath)
		if err != nil {
			return fmt.Errorf("failed to create zip entry for %s: %w", doc.ChaveAcesso, err)
		}

		if _, err := writer.Write(data); err != nil {
			return fmt.Errorf("failed to write file %s to zip: %w", doc.ChaveAcesso, err)
		}
	}

	return nil
}

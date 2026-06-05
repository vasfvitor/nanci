package report

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestGenerateZIPUsesCompanyRoleFolders(t *testing.T) {
	baseDir := t.TempDir()
	xmlRelPath := filepath.Join("xml", "2026-06", "NFSZIP.xml")
	xmlFullPath := filepath.Join(baseDir, xmlRelPath)
	if err := os.MkdirAll(filepath.Dir(xmlFullPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(xmlFullPath, []byte("<NFSe/>"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	outPath := filepath.Join(baseDir, "docs.zip")
	documents := []nfse.CompanyDocument{
		{
			Document: nfse.Document{
				ChaveAcesso: "NFSZIP",
				Competence:  "2026-06",
				XMLPath:     xmlRelPath,
			},
			CompanyRole: "none",
		},
	}

	if err := GenerateZIP(documents, baseDir, outPath); err != nil {
		t.Fatalf("GenerateZIP: %v", err)
	}

	archive, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	defer archive.Close()

	if len(archive.File) != 1 {
		t.Fatalf("expected 1 zip entry, got %d", len(archive.File))
	}
	if archive.File[0].Name != "2026-06/sem-papel-fiscal/NFSZIP.xml" {
		t.Fatalf("unexpected zip entry path: %s", archive.File[0].Name)
	}
}

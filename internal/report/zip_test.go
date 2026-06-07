package report

import (
	"archive/zip"
	"path/filepath"
	"testing"

	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestGenerateZIPUsesCompanyRoleFolders(t *testing.T) {
	baseDir := t.TempDir()
	store := files.NewBlobStore(baseDir)

	err := store.Store("hash123", []byte("<NFSe/>"))
	if err != nil {
		t.Fatalf("Store: %v", err)
	}

	outPath := filepath.Join(baseDir, "docs.zip")
	documents := []nfse.CompanyDocument{
		{
			Document: nfse.Document{
				ChaveAcesso: "NFSZIP",
				Competence:  "2026-06",
				RawHash:     "hash123",
			},
			CompanyRole: "none",
		},
	}

	if err := GenerateZIP(BuildRows(documents), store, outPath); err != nil {
		t.Fatalf("GenerateZIP: %v", err)
	}

	archive, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	defer func() {
		if err := archive.Close(); err != nil {
			t.Errorf("close archive: %v", err)
		}
	}()

	if len(archive.File) != 1 {
		t.Fatalf("expected 1 zip entry, got %d", len(archive.File))
	}
	if archive.File[0].Name != "2026-06/sem-papel-fiscal/NFSZIP.xml" {
		t.Fatalf("unexpected zip entry path: %s", archive.File[0].Name)
	}
}

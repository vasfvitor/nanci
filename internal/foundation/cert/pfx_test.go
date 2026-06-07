package cert

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPKCS12_FileNotFound(t *testing.T) {
	_, err := LoadPKCS12("non_existent_file.pfx", "password")
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("expected ErrFileNotFound, got %v", err)
	}
}

func TestLoadPKCS12_InvalidData(t *testing.T) {
	tempDir := t.TempDir()
	invalidFile := filepath.Join(tempDir, "invalid.pfx")

	err := os.WriteFile(invalidFile, []byte("this is not a valid pfx"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadPKCS12(invalidFile, "password")
	if err == nil {
		t.Error("expected error for invalid PFX data, got nil")
	}
}

func TestLoadPKCS12_ValidMockCert(t *testing.T) {
	mockPfxPath := filepath.Join("testdata", "cert_a1_mock_70860312000150.pfx")
	if _, err := os.Stat(mockPfxPath); os.IsNotExist(err) {
		t.Skip("Mock cert not found, skipping. Run 'go run gen/mock_cert.go' to generate it.")
	}

	loaded, err := LoadPKCS12(mockPfxPath, "mockdata")
	if err != nil {
		t.Fatalf("LoadPKCS12 failed on valid mock cert: %v", err)
	}

	if loaded.TLS.PrivateKey == nil {
		t.Error("expected PrivateKey to be populated, got nil")
	}

	if loaded.Inspection.OwnerCNPJ != "70860312000150" {
		t.Errorf("expected OwnerCNPJ to be 70860312000150, got %q", loaded.Inspection.OwnerCNPJ)
	}

	if loaded.Inspection.OwnerCNPJRoot != "70860312" {
		t.Errorf("expected OwnerCNPJRoot to be 70860312, got %q", loaded.Inspection.OwnerCNPJRoot)
	}

	if loaded.Inspection.FingerprintSHA256 == "" {
		t.Error("expected non-empty FingerprintSHA256")
	}
}

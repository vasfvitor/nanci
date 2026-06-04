package cert

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPKCS12_FileNotFound(t *testing.T) {
	_, err := LoadPKCS12("non_existent_file.pfx", "password")
	if err != ErrFileNotFound {
		t.Errorf("expected ErrFileNotFound, got %v", err)
	}
}

func TestLoadPKCS12_InvalidData(t *testing.T) {
	tempDir := t.TempDir()
	invalidFile := filepath.Join(tempDir, "invalid.pfx")

	err := os.WriteFile(invalidFile, []byte("this is not a valid pfx"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = LoadPKCS12(invalidFile, "password")
	if err == nil {
		t.Error("expected error for invalid PFX data, got nil")
	}
}

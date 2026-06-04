package paths

import (
	"os"
	"strings"
	"testing"
)

func TestDataDir(t *testing.T) {
	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() error = %v", err)
	}

	if dir == "" {
		t.Error("DataDir() returned empty string")
	}

	if !strings.Contains(dir, "nanci") {
		t.Errorf("DataDir() should contain 'nanci', got: %s", dir)
	}
}

func TestEnsureDir(t *testing.T) {
	tempDir := t.TempDir()
	testPath := tempDir + "/test_ensure_dir"

	err := EnsureDir(testPath)
	if err != nil {
		t.Fatalf("EnsureDir() error = %v", err)
	}

	info, err := os.Stat(testPath)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}

	if !info.IsDir() {
		t.Errorf("EnsureDir() did not create a directory")
	}
}

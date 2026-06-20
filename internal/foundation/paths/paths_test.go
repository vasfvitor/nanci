package paths

import (
	"errors"
	"os"
	"path/filepath"
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

func TestDefaultDataDirUsesLocalAppDataOnWindows(t *testing.T) {
	t.Parallel()

	dir, err := defaultDataDir(
		"windows",
		func(key string) string {
			if key == "LOCALAPPDATA" {
				return `C:\Users\alice\AppData\Local`
			}
			return ""
		},
		func() (string, error) {
			t.Fatal("userConfigDir should not be called when LOCALAPPDATA is set")
			return "", nil
		},
		func() (string, error) {
			t.Fatal("userHomeDir should not be called when LOCALAPPDATA is set")
			return "", nil
		},
	)
	if err != nil {
		t.Fatalf("defaultDataDir() error = %v", err)
	}

	want := filepath.Join(`C:\Users\alice\AppData\Local`, "nanci")
	if dir != want {
		t.Fatalf("defaultDataDir() = %q, want %q", dir, want)
	}
}

func TestDefaultDataDirFallsBackToUserConfigDir(t *testing.T) {
	t.Parallel()

	dir, err := defaultDataDir(
		"linux",
		func(string) string { return "" },
		func() (string, error) { return "/home/alice/.config", nil },
		func() (string, error) {
			t.Fatal("userHomeDir should not be called when userConfigDir succeeds")
			return "", nil
		},
	)
	if err != nil {
		t.Fatalf("defaultDataDir() error = %v", err)
	}

	want := filepath.Join("/home/alice/.config", "nanci")
	if dir != want {
		t.Fatalf("defaultDataDir() = %q, want %q", dir, want)
	}
}

func TestDefaultDataDirFallsBackToHomeDir(t *testing.T) {
	t.Parallel()

	dir, err := defaultDataDir(
		"linux",
		func(string) string { return "" },
		func() (string, error) { return "", errors.New("boom") },
		func() (string, error) { return "/home/alice", nil },
	)
	if err != nil {
		t.Fatalf("defaultDataDir() error = %v", err)
	}

	want := filepath.Join("/home/alice", ".nanci")
	if dir != want {
		t.Fatalf("defaultDataDir() = %q, want %q", dir, want)
	}
}

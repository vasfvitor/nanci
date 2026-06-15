package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotatingFileWriterRotatesAndPreservesBackups(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "desktop.log")

	writer, err := newRotatingFileWriter(basePath, 16, 2)
	if err != nil {
		t.Fatalf("newRotatingFileWriter: %v", err)
	}
	t.Cleanup(func() {
		_ = writer.Close()
	})

	for _, line := range []string{"first line\n", "second line\n", "third line\n"} {
		if _, err := writer.Write([]byte(line)); err != nil {
			t.Fatalf("writer.Write(%q): %v", line, err)
		}
	}

	currentContent, err := os.ReadFile(basePath)
	if err != nil {
		t.Fatalf("ReadFile(current): %v", err)
	}
	if !strings.Contains(string(currentContent), "third line") {
		t.Fatalf("current log = %q, want it to contain latest entry", string(currentContent))
	}

	firstBackup, err := os.ReadFile(rotatedLogPath(basePath, 1))
	if err != nil {
		t.Fatalf("ReadFile(backup1): %v", err)
	}
	if !strings.Contains(string(firstBackup), "second line") {
		t.Fatalf("backup1 = %q, want it to contain previous entry", string(firstBackup))
	}

	secondBackup, err := os.ReadFile(rotatedLogPath(basePath, 2))
	if err != nil {
		t.Fatalf("ReadFile(backup2): %v", err)
	}
	if !strings.Contains(string(secondBackup), "first line") {
		t.Fatalf("backup2 = %q, want it to contain oldest retained entry", string(secondBackup))
	}
}

func TestCollectRotatedLogPathsReturnsOldestToNewest(t *testing.T) {
	t.Parallel()

	paths := collectRotatedLogPaths("app.log", 3)
	want := []string{"app.log.3", "app.log.2", "app.log.1", "app.log"}

	if len(paths) != len(want) {
		t.Fatalf("len(paths) = %d, want %d", len(paths), len(want))
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("paths[%d] = %q, want %q", i, paths[i], want[i])
		}
	}
}

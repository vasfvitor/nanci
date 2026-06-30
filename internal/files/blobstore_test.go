package files

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBlobStore_Store_EmptyHash(t *testing.T) {
	store := NewBlobStore(t.TempDir())
	err := store.Store("", []byte("data"))
	if err == nil {
		t.Fatal("expected error for empty hash, got nil")
	}
}

func TestBlobStore_Get_EmptyHash(t *testing.T) {
	store := NewBlobStore(t.TempDir())
	_, err := store.Get("")
	if err == nil {
		t.Fatal("expected error for empty hash, got nil")
	}
}

func TestBlobStore_Get_NotFound(t *testing.T) {
	store := NewBlobStore(t.TempDir())
	_, err := store.Get("nonexistent-hash")
	if !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("expected ErrFileNotFound, got %v", err)
	}
}

func TestBlobStore_Store_CreatesDirAndFile(t *testing.T) {
	base := t.TempDir()
	store := NewBlobStore(base)

	hash := "abc123"
	data := []byte("<nfse>xml content</nfse>")

	if err := store.Store(hash, data); err != nil {
		t.Fatalf("Store returned unexpected error: %v", err)
	}

	// Verify file exists at expected path with .xml extension.
	expectedPath := filepath.Join(base, "blobs", hash+".xml")
	got, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("expected file at %s, got error: %v", expectedPath, err)
	}
	if string(got) != string(data) {
		t.Errorf("file content = %q, want %q", got, data)
	}
}

func TestBlobStore_Store_DirAlreadyExists(t *testing.T) {
	base := t.TempDir()
	store := NewBlobStore(base)

	// Pre-create the blobs directory.
	blobsDir := filepath.Join(base, "blobs")
	if err := os.MkdirAll(blobsDir, 0o750); err != nil {
		t.Fatalf("setup: %v", err)
	}

	hash := "def456"
	data := []byte("second store")

	if err := store.Store(hash, data); err != nil {
		t.Fatalf("Store returned unexpected error when dir exists: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(blobsDir, hash+".xml"))
	if err != nil {
		t.Fatalf("file missing after store: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("content = %q, want %q", got, data)
	}
}

func TestBlobStore_RoundTrip(t *testing.T) {
	store := NewBlobStore(t.TempDir())

	hash := "roundtrip-hash"
	data := []byte("<xml>round trip data</xml>")

	if err := store.Store(hash, data); err != nil {
		t.Fatalf("Store: %v", err)
	}

	got, err := store.Get(hash)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("round-trip mismatch: got %q, want %q", got, data)
	}
}

func TestBlobStore_MultipleHashesAreIndependent(t *testing.T) {
	store := NewBlobStore(t.TempDir())

	pairs := []struct {
		hash string
		data []byte
	}{
		{"hash-alpha", []byte("alpha content")},
		{"hash-beta", []byte("beta content")},
		{"hash-gamma", []byte("gamma content")},
	}

	for _, p := range pairs {
		if err := store.Store(p.hash, p.data); err != nil {
			t.Fatalf("Store(%q): %v", p.hash, err)
		}
	}

	for _, p := range pairs {
		got, err := store.Get(p.hash)
		if err != nil {
			t.Fatalf("Get(%q): %v", p.hash, err)
		}
		if string(got) != string(p.data) {
			t.Errorf("Get(%q) = %q, want %q", p.hash, got, p.data)
		}
	}
}

func TestBlobStore_Store_WriteError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("read-only directory semantics differ on Windows")
	}

	base := t.TempDir()
	store := NewBlobStore(base)

	// Pre-create the blobs directory, then make it read-only.
	blobsDir := filepath.Join(base, "blobs")
	if err := os.MkdirAll(blobsDir, 0o750); err != nil {
		t.Fatalf("setup MkdirAll: %v", err)
	}
	if err := os.Chmod(blobsDir, 0o444); err != nil {
		t.Fatalf("setup Chmod: %v", err)
	}
	// Restore permissions so t.TempDir cleanup can delete the dir.
	t.Cleanup(func() { os.Chmod(blobsDir, 0o755) }) //nolint:errcheck

	err := store.Store("somehash", []byte("data"))
	if err == nil {
		t.Fatal("expected error writing to read-only dir, got nil")
	}
}

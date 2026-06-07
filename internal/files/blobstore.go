package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrFileNotFound = errors.New("blob not found")

// XMLStore defines the interface for storing and retrieving raw XML files by their hash.
type XMLStore interface {
	Store(hash string, data []byte) error
	Get(hash string) ([]byte, error)
}

// BlobStore implements XMLStore, storing files in a flat structure using the SHA-256 hash as the filename.
type BlobStore struct {
	blobsDir string
}

// NewBlobStore creates a new BlobStore. baseDir should be the root data directory.
func NewBlobStore(baseDir string) *BlobStore {
	return &BlobStore{
		blobsDir: filepath.Join(baseDir, "blobs"),
	}
}

// Store saves the data to the blob store using the provided hash as the filename.
func (b *BlobStore) Store(hash string, data []byte) error {
	if hash == "" {
		return errors.New("hash cannot be empty")
	}

	if err := os.MkdirAll(b.blobsDir, 0750); err != nil {
		return fmt.Errorf("failed to create blobs directory: %w", err)
	}

	fullPath := filepath.Join(b.blobsDir, hash+".xml")

	if err := os.WriteFile(fullPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write blob file: %w", err)
	}

	return nil
}

// Get retrieves the data from the blob store by hash.
func (b *BlobStore) Get(hash string) ([]byte, error) {
	if hash == "" {
		return nil, errors.New("hash cannot be empty")
	}

	fullPath := filepath.Join(b.blobsDir, hash+".xml")
	data, err := os.ReadFile(fullPath) // #nosec G304 -- fullPath is rooted in the configured blob directory.
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("failed to read blob file: %w", err)
	}

	return data, nil
}

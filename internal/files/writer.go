package files

import (
	"fmt"
	"os"
	"path/filepath"
)

// Writer is responsible for storing XML files on disk.
type Writer struct {
	baseDir string
}

// NewWriter creates a new Writer. baseDir is the root directory for data (e.g., DataDir).
func NewWriter(baseDir string) *Writer {
	return &Writer{
		baseDir: baseDir,
	}
}

// SaveXML saves the raw XML data to disk, organized by CNPJ, Competence, and Direction.
// It returns the relative path to the saved file.
func (w *Writer) SaveXML(cnpj string, competence string, direction string, chaveAcesso string, data []byte) (string, error) {
	if competence == "" {
		competence = "0000-00" // Fallback if competence is missing
	}
	if direction == "" {
		direction = "desconhecido"
	}

	// Create directory structure: <baseDir>/xml/<cnpj>/<competence>/<direction>
	relDir := filepath.Join("xml", cnpj, competence, direction)
	fullDir := filepath.Join(w.baseDir, relDir)

	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	fileName := fmt.Sprintf("%s.xml", chaveAcesso)
	relPath := filepath.Join(relDir, fileName)
	fullPath := filepath.Join(fullDir, fileName)

	// Save file
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write xml file: %w", err)
	}

	return relPath, nil
}

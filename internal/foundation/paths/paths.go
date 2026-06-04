package paths

import (
	"os"
	"path/filepath"
)

// DataDir returns the base directory for nanci data storage.
// In normal environments, it uses os.UserConfigDir() (or LocalAppData).
// If an error occurs or if specified, it may return a fallback.
func DataDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback if unable to get the OS config dir
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, ".nanci"), nil
	}
	return filepath.Join(configDir, "nanci"), nil
}

// EnsureDir ensures the specified directory exists, creating it if necessary.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

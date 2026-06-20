package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// DataDir returns the base directory for nanci data storage.
// On Windows, it prefers LocalAppData to keep desktop installs and app data
// under the per-user local profile. On other platforms, it uses UserConfigDir.
// If an error occurs or if specified, it may return a fallback.
func DataDir() (string, error) {
	if envDir := os.Getenv("NANCI_DATA_DIR"); envDir != "" {
		return envDir, nil
	}

	return defaultDataDir(runtime.GOOS, os.Getenv, os.UserConfigDir, os.UserHomeDir)
}

// EnsureDir ensures the specified directory exists, creating it if necessary.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o750)
}

func defaultDataDir(
	goos string,
	getenv func(string) string,
	userConfigDir func() (string, error),
	userHomeDir func() (string, error),
) (string, error) {
	if goos == "windows" {
		if localAppData := getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "nanci"), nil
		}
	}

	configDir, err := userConfigDir()
	if err == nil {
		return filepath.Join(configDir, "nanci"), nil
	}

	homeDir, homeErr := userHomeDir()
	if homeErr != nil {
		return "", homeErr
	}
	return filepath.Join(homeDir, ".nanci"), nil
}

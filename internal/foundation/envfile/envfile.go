package envfile

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const fileName = ".env.local"

// LoadLocal loads variables from supported .env.local locations.
// Existing process environment variables always win.
func LoadLocal() error {
	for _, path := range candidatePaths() {
		if err := loadFile(path); err != nil {
			return err
		}
	}
	return nil
}

func candidatePaths() []string {
	seen := make(map[string]struct{})
	var paths []string

	add := func(path string) {
		if path == "" {
			return
		}
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			return
		}
		seen[clean] = struct{}{}
		paths = append(paths, clean)
	}

	if cwd, err := os.Getwd(); err == nil {
		add(filepath.Join(cwd, fileName))
	}
	if exe, err := os.Executable(); err == nil {
		add(filepath.Join(filepath.Dir(exe), fileName))
	}
	if configDir, err := os.UserConfigDir(); err == nil {
		add(filepath.Join(configDir, "nanci", fileName))
	}

	return paths
}

func loadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		key, value, ok, err := parseLine(scanner.Text())
		if err != nil {
			return fmt.Errorf("parse %s:%d: %w", path, lineNo, err)
		}
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s from %s:%d: %w", key, path, lineNo, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	return nil
}

func parseLine(line string) (key, value string, ok bool, err error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false, nil
	}
	if strings.HasPrefix(trimmed, "export ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
	}

	key, value, found := strings.Cut(trimmed, "=")
	if !found {
		return "", "", false, fmt.Errorf("missing '='")
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", false, fmt.Errorf("empty key")
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return key, "", true, nil
	}

	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", "", false, fmt.Errorf("invalid quoted value: %w", err)
		}
		return key, unquoted, true, nil
	}
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return key, value[1 : len(value)-1], true, nil
	}

	if idx := strings.Index(value, " #"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	return key, value, true, nil
}

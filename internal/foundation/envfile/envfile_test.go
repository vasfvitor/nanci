package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLocalLoadsCurrentDirectoryFile(t *testing.T) {
	oldValue, hadValue := os.LookupEnv("NANCI_CERT_PASSWORD")
	if err := os.Unsetenv("NANCI_CERT_PASSWORD"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if !hadValue {
			_ = os.Unsetenv("NANCI_CERT_PASSWORD")
			return
		}
		_ = os.Setenv("NANCI_CERT_PASSWORD", oldValue)
	})

	dir := t.TempDir()
	path := filepath.Join(dir, fileName)
	if err := os.WriteFile(path, []byte("NANCI_CERT_PASSWORD=from-file\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if err := LoadLocal(); err != nil {
		t.Fatalf("LoadLocal() error = %v", err)
	}

	if got := os.Getenv("NANCI_CERT_PASSWORD"); got != "from-file" {
		t.Fatalf("NANCI_CERT_PASSWORD = %q, want %q", got, "from-file")
	}
}

func TestLoadLocalDoesNotOverrideExistingEnv(t *testing.T) {
	t.Setenv("NANCI_CERT_PASSWORD", "from-env")

	dir := t.TempDir()
	path := filepath.Join(dir, fileName)
	if err := os.WriteFile(path, []byte("NANCI_CERT_PASSWORD=from-file\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if err := LoadLocal(); err != nil {
		t.Fatalf("LoadLocal() error = %v", err)
	}

	if got := os.Getenv("NANCI_CERT_PASSWORD"); got != "from-env" {
		t.Fatalf("NANCI_CERT_PASSWORD = %q, want %q", got, "from-env")
	}
}

func TestParseLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantKey   string
		wantValue string
		wantOK    bool
		wantErr   bool
	}{
		{name: "blank", line: "   ", wantOK: false},
		{name: "comment", line: "# comment", wantOK: false},
		{name: "plain", line: "FOO=bar", wantKey: "FOO", wantValue: "bar", wantOK: true},
		{name: "quoted double", line: "FOO=\"bar baz\"", wantKey: "FOO", wantValue: "bar baz", wantOK: true},
		{name: "quoted single", line: "FOO='bar baz'", wantKey: "FOO", wantValue: "bar baz", wantOK: true},
		{name: "export", line: "export FOO=bar", wantKey: "FOO", wantValue: "bar", wantOK: true},
		{name: "inline comment", line: "FOO=bar # note", wantKey: "FOO", wantValue: "bar", wantOK: true},
		{name: "missing equals", line: "FOO", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKey, gotValue, gotOK, err := parseLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseLine() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if gotKey != tt.wantKey || gotValue != tt.wantValue || gotOK != tt.wantOK {
				t.Fatalf("parseLine() = (%q, %q, %v), want (%q, %q, %v)", gotKey, gotValue, gotOK, tt.wantKey, tt.wantValue, tt.wantOK)
			}
		})
	}
}

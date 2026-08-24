package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeLogContent(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "raw numeric", in: "cnpj=12345678000195 ok", want: "cnpj=12.***.***/****-95 ok"},
		{name: "formatted numeric", in: "12.345.678/0001-95", want: "12.***.***/****-95"},
		{name: "formatted alphanumeric", in: "empresa 12.ABC.345/01DE-35", want: "empresa 12.***.***/****-35"},
		{name: "query param", in: `url="DFe/1?cnpjConsulta=12345678000195&lote=true"`, want: `url="DFe/1?cnpjConsulta=12.***.***/****-95&lote=true"`},
		{name: "two per line", in: "12345678000195 -> 98765432000121", want: "12.***.***/****-95 -> 98.***.***/****-21"},
		{name: "hex id of 14 chars untouched", in: "id=3fa9c1d2e4b5a6", want: "id=3fa9c1d2e4b5a6"},
		{name: "raw alphanumeric untouched", in: "12ABC34501DE35", want: "12ABC34501DE35"},
		{name: "access key untouched", in: "chave=35503082604800000000000000000000000000000000000001", want: "chave=35503082604800000000000000000000000000000000000001"},
		{name: "13 digits untouched", in: "1234567800019", want: "1234567800019"},
		{name: "15 digits untouched", in: "123456780001951", want: "123456780001951"},
		{name: "empty", in: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(sanitizeLogContent([]byte(tt.in))); got != tt.want {
				t.Errorf("sanitizeLogContent(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestExportRotatedLogsMasksCNPJs(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "nanci-desktop.log")
	raw := "sync company cnpj=12345678000195 url=\"DFe/1?cnpjConsulta=12345678000195\"\n"
	if err := os.WriteFile(logPath, []byte(raw), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}

	zipPath := filepath.Join(dir, "out.zip")
	if err := exportRotatedLogs(zipPath, logPath); err != nil {
		t.Fatalf("exportRotatedLogs: %v", err)
	}

	onDisk, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if string(onDisk) != raw {
		t.Fatalf("log on disk was modified: %q", onDisk)
	}

	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer func() { _ = archive.Close() }()
	if len(archive.File) != 1 {
		t.Fatalf("zip has %d entries, want 1", len(archive.File))
	}
	entry, err := archive.File[0].Open()
	if err != nil {
		t.Fatalf("open entry: %v", err)
	}
	defer func() { _ = entry.Close() }()
	exported, err := io.ReadAll(entry)
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	if strings.Contains(string(exported), "12345678000195") {
		t.Fatalf("exported log leaks CNPJ: %q", exported)
	}
	if strings.Count(string(exported), "12.***.***/****-95") != 2 {
		t.Fatalf("expected both CNPJs masked, got %q", exported)
	}
}

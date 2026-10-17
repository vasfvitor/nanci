package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestShowInFolderCommand(t *testing.T) {
	file := filepath.Join("saida", "notas.zip")
	dir := "saida"
	tests := []struct {
		name     string
		goos     string
		path     string
		isDir    bool
		wantName string
		wantArgs []string
	}{
		{"windows/file", "windows", file, false, "explorer", []string{"/select," + file}},
		{"windows/dir", "windows", dir, true, "explorer", []string{dir}},
		{"darwin/file", "darwin", file, false, "open", []string{"-R", file}},
		{"darwin/dir", "darwin", dir, true, "open", []string{"-R", dir}},
		{"linux/file", "linux", file, false, "xdg-open", []string{filepath.Dir(file)}},
		{"linux/dir", "linux", dir, true, "xdg-open", []string{dir}},
		{"freebsd/file", "freebsd", file, false, "xdg-open", []string{filepath.Dir(file)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, args := showInFolderCommand(tt.goos, tt.path, tt.isDir)
			if name != tt.wantName || !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("showInFolderCommand(%q, %q, %v) = %q %v, want %q %v",
					tt.goos, tt.path, tt.isDir, name, args, tt.wantName, tt.wantArgs)
			}
		})
	}
}

func TestShowInFolderRejectsMissingFile(t *testing.T) {
	a := newTestApp(services{})
	err := a.ShowInFolder(filepath.Join(t.TempDir(), "sumiu.zip"))
	if err == nil || !strings.Contains(err.Error(), "arquivo não encontrado") {
		t.Errorf("err = %v, want arquivo não encontrado", err)
	}
}

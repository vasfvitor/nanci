package main

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// frontendPackages are the Go packages whose types may reach the frontend.
// The Wails generator writes one folder under frontend/bindings for every
// package a bound method or a registered event carries, field types
// included, so the folder list is the boundary of AGENTS.md: screens get
// desktopapi DTOs, not structs of internal/app, internal/nfe or any other
// core package. A new folder means a core type crossed over; add a DTO and a
// mapper in desktopapi instead (see the desktop-method skill).
var frontendPackages = []string{
	"events", // the event name to payload map the generator writes
	"github.com/vasfvitor/nanci/internal/desktop",
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi",
	// The request-cert-password event still carries sync.CertPasswordRequest
	// (events.go); moving it to a desktopapi DTO removes this line.
	"github.com/vasfvitor/nanci/internal/sync",
	"github.com/wailsapp/wails/v3/internal",
	"github.com/wailsapp/wails/v3/pkg/updater",
	"time",
}

// TestOnlyDesktopAPITypesReachTheFrontend reads the committed bindings,
// which scripts/verify.sh and CI keep in step with the Go code.
func TestOnlyDesktopAPITypesReachTheFrontend(t *testing.T) {
	root := filepath.Join("frontend", "bindings")
	var found []string
	err := fs.WalkDir(os.DirFS(root), ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(name, ".ts") {
			found = append(found, path.Dir(name))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("read %s: %v", root, err)
	}
	slices.Sort(found)
	found = slices.Compact(found)
	if len(found) == 0 {
		t.Fatalf("no bindings under %s", root)
	}
	for _, pkg := range found {
		if !slices.Contains(frontendPackages, pkg) {
			t.Errorf("frontend/bindings/%s: types of %s reach the frontend; bound methods and events carry desktopapi DTOs", pkg, pkg)
		}
	}
}

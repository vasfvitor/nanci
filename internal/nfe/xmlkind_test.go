package nfe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vasfvitor/nanci/internal/dfe"
)

func TestKindOfRootForEveryFixture(t *testing.T) {
	// Every XML in testdata must be listed; invalid.xml has no root.
	want := map[string]XMLKind{
		"proceventonfe-cancelamento.xml": KindProcEventoNFe,
		"proceventonfe-cce.xml":          KindProcEventoNFe,
		"proceventonfe-ciencia-real.xml": KindProcEventoNFe,
		"proceventonfe-ciencia.xml":      KindProcEventoNFe,
		"procnfe-alfanumerico.xml":       KindProcNFe,
		"procnfe-denegada.xml":           KindProcNFe,
		"procnfe.xml":                    KindProcNFe,
		"resevento-cancelamento.xml":     KindResEvento,
		"resnfe-cancelada.xml":           KindResNFe,
		"resnfe.xml":                     KindResNFe,
		"invalid.xml":                    "",
	}
	files, err := filepath.Glob(filepath.Join("testdata", "*.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != len(want) {
		t.Errorf("testdata has %d XML files, the table lists %d", len(files), len(want))
	}
	for _, path := range files {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			wantKind, listed := want[name]
			if !listed {
				t.Fatalf("%s is not in the table", name)
			}
			data, err := os.ReadFile(path) // #nosec G304 -- testdata path.
			if err != nil {
				t.Fatal(err)
			}
			root, err := dfe.RootElement(data)
			if wantKind == "" {
				if err == nil {
					t.Errorf("RootElement = %q, want an error", root)
				}
				return
			}
			if err != nil {
				t.Fatalf("RootElement: %v", err)
			}
			kind, ok := KindOfRoot(root)
			if !ok || kind != wantKind {
				t.Fatalf("KindOfRoot(%q) = %q, %v; want %q", root, kind, ok, wantKind)
			}
			// The kind names the parser that reads the file.
			switch kind {
			case KindProcNFe:
				_, err = ParseProcNFe(data)
			case KindResNFe:
				_, err = ParseResNFe(data)
			case KindProcEventoNFe:
				_, err = ParseProcEventoNFe(data)
			case KindResEvento:
				_, err = ParseResEvento(data)
			}
			if err != nil {
				t.Errorf("parse as %s: %v", kind, err)
			}
		})
	}
}

func TestKindOfRootRejectsOtherRoots(t *testing.T) {
	for _, root := range []string{"NFe", "cteProc", "procEventoCTe", "retDistDFeInt", "", "nfeproc"} {
		if kind, ok := KindOfRoot(root); ok {
			t.Errorf("KindOfRoot(%q) = %q, want not ok", root, kind)
		}
	}
}

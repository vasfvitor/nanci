package cte

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vasfvitor/nanci/internal/dfe"
)

func TestKindOfRootForEveryFixture(t *testing.T) {
	// Every XML in testdata must be listed; invalid.xml has no root.
	want := map[string]XMLKind{
		"proceventocte-cancelamento.xml": KindProcEventoCTe,
		"proceventocte-comprovante.xml":  KindProcEventoCTe,
		"procgtve.xml":                   KindProcCTe,
		"procte-toma4.xml":               KindProcCTe,
		"procte-v200-toma03.xml":         KindProcCTe,
		"procte.xml":                     KindProcCTe,
		"procteos.xml":                   KindProcCTe,
		"proctesimp.xml":                 KindProcCTe,
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
			case KindProcCTe:
				_, err = ParseProcCTe(data)
			case KindProcEventoCTe:
				_, err = ParseProcEventoCTe(data)
			}
			if err != nil {
				t.Errorf("parse as %s: %v", kind, err)
			}
		})
	}
}

func TestKindOfRootRejectsOtherRoots(t *testing.T) {
	for _, root := range []string{"CTe", "nfeProc", "procEventoNFe", "retDistDFeInt", "", "cteproc"} {
		if kind, ok := KindOfRoot(root); ok {
			t.Errorf("KindOfRoot(%q) = %q, want not ok", root, kind)
		}
	}
}

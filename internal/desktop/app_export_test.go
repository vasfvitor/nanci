package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
)

var errCore = errors.New("falha no core")

const (
	testCNPJ  = "12345678000195"
	testChave = "35260912345678000195550010000000011000000010"
	testOut   = "C:/exports/saida"
)

func TestExportDocumentsFormats(t *testing.T) {
	tests := []struct {
		format string
		method string
	}{
		{"csv", "ExportCSV"},
		{"xlsx", "ExportXLSX"},
		{"zip", "ExportZIP"},
		{" XLSX ", "ExportXLSX"},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			f, a := newFakeApp(nil)
			f.exports.result = app.ExportResult{OutPath: testOut + ".csv", Format: "csv", Incremental: true, ExportedCount: 3}

			got, err := a.ExportDocuments(desktopapi.ExportDocumentsInput{
				CNPJ:         testCNPJ,
				Competence:   "2026-09",
				Direction:    "tomada",
				Format:       tt.format,
				OutPath:      testOut,
				Incremental:  true,
				ChavesAcesso: []string{"chave-1", "chave-2"},
			})
			if err != nil {
				t.Fatalf("ExportDocuments: %v", err)
			}

			assertCalls(t, &f.exports.recorder, call{tt.method, []any{app.ExportInput{
				CNPJ:         testCNPJ,
				Competence:   "2026-09",
				Direction:    "tomada",
				OutPath:      testOut,
				Incremental:  true,
				ChavesAcesso: []string{"chave-1", "chave-2"},
			}}})
			want := desktopapi.ExportResult{OutPath: testOut + ".csv", Format: "csv", Incremental: true, ExportedCount: 3}
			if got != want {
				t.Errorf("result = %+v, want %+v", got, want)
			}
		})
	}
}

func TestExportDocumentsRejectsBeforeCore(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		outPath string
		wantErr string
	}{
		{"unknown format", "pdf", testOut, "formato de exportação desconhecido: pdf"},
		{"empty format", "", testOut, "formato de exportação desconhecido: "},
		{"empty out path", "csv", "", "caminho de saída não especificado"},
		// The path check runs first.
		{"both", "pdf", "", "caminho de saída não especificado"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, a := newFakeApp(nil)

			got, err := a.ExportDocuments(desktopapi.ExportDocumentsInput{CNPJ: testCNPJ, Format: tt.format, OutPath: tt.outPath})
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("err = %v, want %q", err, tt.wantErr)
			}
			if got != (desktopapi.ExportResult{}) {
				t.Errorf("result = %+v, want zero", got)
			}
			f.assertNoCalls(t)
		})
	}
}

// TestSingleDocumentExports pins the current result of the one-document
// exports: OutPath and Format from the request, ExportedCount 0 even though
// one file was written.
func TestSingleDocumentExports(t *testing.T) {
	tests := []struct {
		name     string
		run      func(a *App) (desktopapi.ExportResult, error)
		recorder func(f fakeSet) *recorder
		want     call
		result   desktopapi.ExportResult
	}{
		{
			name: "ExportDANFSe",
			run: func(a *App) (desktopapi.ExportResult, error) {
				return a.ExportDANFSe(desktopapi.ExportDANFSeInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: testOut + ".pdf"})
			},
			recorder: func(f fakeSet) *recorder { return &f.exports.recorder },
			want:     call{"ExportDANFSe", []any{app.ExportDANFSeInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: testOut + ".pdf"}}},
			result:   desktopapi.ExportResult{OutPath: testOut + ".pdf", Format: "danfse"},
		},
		{
			name: "ExportXML",
			run: func(a *App) (desktopapi.ExportResult, error) {
				return a.ExportXML(desktopapi.ExportXMLInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: testOut + ".xml"})
			},
			recorder: func(f fakeSet) *recorder { return &f.exports.recorder },
			want:     call{"ExportXML", []any{app.ExportXMLInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: testOut + ".xml"}}},
			result:   desktopapi.ExportResult{OutPath: testOut + ".xml", Format: "xml"},
		},
		{
			name: "ExportNFeXML",
			run: func(a *App) (desktopapi.ExportResult, error) {
				return a.ExportNFeXML(desktopapi.ExportNFeXMLInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: testOut + ".xml"})
			},
			recorder: func(f fakeSet) *recorder { return &f.nfe.recorder },
			want:     call{"ExportXML", []any{app.NFeExportXMLInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: testOut + ".xml"}}},
			result:   desktopapi.ExportResult{OutPath: testOut + ".xml", Format: "xml"},
		},
		{
			name: "ExportCTeXML",
			run: func(a *App) (desktopapi.ExportResult, error) {
				return a.ExportCTeXML(desktopapi.ExportCTeXMLInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: testOut + ".xml"})
			},
			recorder: func(f fakeSet) *recorder { return &f.cte.recorder },
			want:     call{"ExportXML", []any{app.CTeExportXMLInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: testOut + ".xml"}}},
			result:   desktopapi.ExportResult{OutPath: testOut + ".xml", Format: "xml"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, a := newFakeApp(nil)

			got, err := tt.run(a)
			if err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			assertCalls(t, tt.recorder(f), tt.want)
			if got != tt.result {
				t.Errorf("result = %+v, want %+v", got, tt.result)
			}
		})
	}
}

func TestExportDANFSeZIP(t *testing.T) {
	f, a := newFakeApp(nil)
	f.exports.result = app.ExportResult{OutPath: testOut + ".zip", Format: "danfse", Incremental: true, ExportedCount: 7}

	got, err := a.ExportDANFSeZIP(desktopapi.ExportDocumentsInput{
		CNPJ:         testCNPJ,
		Competence:   "2026-09",
		Direction:    "prestada",
		Format:       "ignored",
		OutPath:      testOut + ".zip",
		Incremental:  true,
		ChavesAcesso: []string{testChave},
	})
	if err != nil {
		t.Fatalf("ExportDANFSeZIP: %v", err)
	}

	assertCalls(t, &f.exports.recorder, call{"ExportDANFSeZIP", []any{app.ExportInput{
		CNPJ:         testCNPJ,
		Competence:   "2026-09",
		Direction:    "prestada",
		OutPath:      testOut + ".zip",
		Incremental:  true,
		ChavesAcesso: []string{testChave},
	}}})
	want := desktopapi.ExportResult{OutPath: testOut + ".zip", Format: "danfse", Incremental: true, ExportedCount: 7}
	if got != want {
		t.Errorf("result = %+v, want %+v", got, want)
	}
}

func TestExportNFeZIP(t *testing.T) {
	f, a := newFakeApp(nil)
	f.nfe.zip = app.NFeExportResult{
		ExportResult:   app.ExportResult{OutPath: testOut + ".zip", Format: "xml", Incremental: true, ExportedCount: 4},
		SkippedResumos: 2,
	}

	got, err := a.ExportNFeZIP(desktopapi.ExportNFeZIPInput{
		CNPJ:           testCNPJ,
		Competence:     "2026-09",
		Role:           "destinatario",
		ChavesAcesso:   []string{testChave},
		IncludeResumos: true,
		Incremental:    true,
		OutPath:        testOut + ".zip",
	})
	if err != nil {
		t.Fatalf("ExportNFeZIP: %v", err)
	}

	assertCalls(t, &f.nfe.recorder, call{"ExportXMLZip", []any{app.NFeExportInput{
		CNPJ:           testCNPJ,
		Competence:     "2026-09",
		Role:           "destinatario",
		ChavesAcesso:   []string{testChave},
		IncludeResumos: true,
		Incremental:    true,
		OutPath:        testOut + ".zip",
	}}})
	want := desktopapi.NFeExportResult{
		ExportResult:   desktopapi.ExportResult{OutPath: testOut + ".zip", Format: "xml", Incremental: true, ExportedCount: 4},
		SkippedResumos: 2,
	}
	if got != want {
		t.Errorf("result = %+v, want %+v", got, want)
	}
}

func TestExportCTeZIP(t *testing.T) {
	f, a := newFakeApp(nil)
	f.cte.zip = app.ExportResult{OutPath: testOut + ".zip", Format: "xml", ExportedCount: 5}

	got, err := a.ExportCTeZIP(desktopapi.ExportCTeZIPInput{
		CNPJ:         testCNPJ,
		Competence:   "2026-09",
		Role:         "tomador",
		ChavesAcesso: []string{testChave},
		OutPath:      testOut + ".zip",
	})
	if err != nil {
		t.Fatalf("ExportCTeZIP: %v", err)
	}

	assertCalls(t, &f.cte.recorder, call{"ExportXMLZip", []any{app.CTeExportInput{
		CNPJ:         testCNPJ,
		Competence:   "2026-09",
		Role:         "tomador",
		ChavesAcesso: []string{testChave},
		OutPath:      testOut + ".zip",
	}}})
	want := desktopapi.ExportResult{OutPath: testOut + ".zip", Format: "xml", ExportedCount: 5}
	if got != want {
		t.Errorf("result = %+v, want %+v", got, want)
	}
}

// exportRuns calls every export method with the given out path.
func exportRuns(outPath string) map[string]func(a *App) (any, error) {
	return map[string]func(a *App) (any, error){
		"ExportDocuments": func(a *App) (any, error) {
			return a.ExportDocuments(desktopapi.ExportDocumentsInput{CNPJ: testCNPJ, Format: "csv", OutPath: outPath})
		},
		"ExportDANFSe": func(a *App) (any, error) {
			return a.ExportDANFSe(desktopapi.ExportDANFSeInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: outPath})
		},
		"ExportXML": func(a *App) (any, error) {
			return a.ExportXML(desktopapi.ExportXMLInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: outPath})
		},
		"ExportDANFSeZIP": func(a *App) (any, error) {
			return a.ExportDANFSeZIP(desktopapi.ExportDocumentsInput{CNPJ: testCNPJ, OutPath: outPath})
		},
		"ExportNFeXML": func(a *App) (any, error) {
			return a.ExportNFeXML(desktopapi.ExportNFeXMLInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: outPath})
		},
		"ExportNFeZIP": func(a *App) (any, error) {
			return a.ExportNFeZIP(desktopapi.ExportNFeZIPInput{CNPJ: testCNPJ, OutPath: outPath})
		},
		"ExportCTeXML": func(a *App) (any, error) {
			return a.ExportCTeXML(desktopapi.ExportCTeXMLInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: outPath})
		},
		"ExportCTeZIP": func(a *App) (any, error) {
			return a.ExportCTeZIP(desktopapi.ExportCTeZIPInput{CNPJ: testCNPJ, OutPath: outPath})
		},
	}
}

func TestExportsRequireOutPath(t *testing.T) {
	for name, run := range exportRuns("") {
		t.Run(name, func(t *testing.T) {
			f, a := newFakeApp(nil)

			got, err := run(a)
			if err == nil || err.Error() != "caminho de saída não especificado" {
				t.Fatalf("err = %v, want the missing out path error", err)
			}
			if !reflect.ValueOf(got).IsZero() {
				t.Errorf("result = %+v, want zero", got)
			}
			f.assertNoCalls(t)
		})
	}
}

func TestExportsPassCoreErrors(t *testing.T) {
	for name, run := range exportRuns(testOut) {
		t.Run(name, func(t *testing.T) {
			_, a := newFakeApp(errCore)

			got, err := run(a)
			if !errors.Is(err, errCore) {
				t.Fatalf("err = %v, want %v", err, errCore)
			}
			if !reflect.ValueOf(got).IsZero() {
				t.Errorf("result = %+v, want zero", got)
			}
		})
	}
}

func TestCountPendingExports(t *testing.T) {
	tests := []struct {
		format string
		kind   string
	}{
		{"zip", "xml"},
		{"danfse-zip", "danfse"},
		{"csv", "csv"},
		{" XLSX ", "xlsx"},
		// Unknown formats reach the core unchanged; it decides.
		{"pdf", "pdf"},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			f, a := newFakeApp(nil)
			f.exports.count = 9

			got, err := a.CountPendingExports(desktopapi.ExportDocumentsInput{
				CNPJ:         testCNPJ,
				Competence:   "2026-09",
				Direction:    "tomada",
				Format:       tt.format,
				OutPath:      testOut,
				Incremental:  true,
				ChavesAcesso: []string{testChave},
			})
			if err != nil {
				t.Fatalf("CountPendingExports: %v", err)
			}
			if got != 9 {
				t.Errorf("count = %d, want 9", got)
			}
			// Only the filters reach the counter; out path, incremental and
			// chaves do not.
			assertCalls(t, &f.exports.recorder, call{"CountPendingExportDocuments", []any{
				app.ExportInput{CNPJ: testCNPJ, Competence: "2026-09", Direction: "tomada"},
				tt.kind,
			}})
		})
	}
}

func TestCountPendingExportsPassesCoreError(t *testing.T) {
	_, a := newFakeApp(errCore)

	if _, err := a.CountPendingExports(desktopapi.ExportDocumentsInput{CNPJ: testCNPJ, Format: "zip"}); !errors.Is(err, errCore) {
		t.Fatalf("err = %v, want %v", err, errCore)
	}
}

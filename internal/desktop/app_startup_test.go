package main

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
)

// coreRuns calls every bound method that needs the core services.
func coreRuns() map[string]func(a *App) (any, error) {
	return map[string]func(a *App) (any, error){
		"AddCompany": func(a *App) (any, error) {
			return nil, a.AddCompany(desktopapi.AddCompanyInput{CNPJ: testCNPJ, Environment: "producao", SyncStartPolicy: "all"})
		},
		"AddCredential": func(a *App) (any, error) {
			return nil, a.AddCredential(desktopapi.AddCredentialInput{Label: "x", CertPath: "x.pfx"})
		},
		"AssignCredentialToCompany": func(a *App) (any, error) {
			return nil, a.AssignCredentialToCompany(desktopapi.AssignCredentialInput{CompanyCNPJ: testCNPJ, CredentialID: "cred-1"})
		},
		"ExportCTeXML": func(a *App) (any, error) {
			return a.ExportCTeXML(context.Background(), desktopapi.ExportCTeXMLInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: testOut})
		},
		"ExportCTeDocuments": func(a *App) (any, error) {
			return a.ExportCTeDocuments(context.Background(), desktopapi.ExportCTeDocumentsInput{CNPJ: testCNPJ, Format: "csv", OutPath: testOut})
		},
		"ExportDACTE": func(a *App) (any, error) {
			return a.ExportDACTE(context.Background(), desktopapi.ExportDACTEInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: testOut})
		},
		"ExportDANFE": func(a *App) (any, error) {
			return a.ExportDANFE(context.Background(), desktopapi.ExportDANFEInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: testOut})
		},
		"ExportDANFSe": func(a *App) (any, error) {
			return a.ExportDANFSe(context.Background(), desktopapi.ExportDANFSeInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: testOut})
		},
		"ExportDocuments": func(a *App) (any, error) {
			return a.ExportDocuments(context.Background(), desktopapi.ExportDocumentsInput{CNPJ: testCNPJ, Format: "csv", OutPath: testOut})
		},
		"ExportNFeXML": func(a *App) (any, error) {
			return a.ExportNFeXML(context.Background(), desktopapi.ExportNFeXMLInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: testOut})
		},
		"ExportNFeDocuments": func(a *App) (any, error) {
			return a.ExportNFeDocuments(context.Background(), desktopapi.ExportNFeDocumentsInput{CNPJ: testCNPJ, Format: "csv", OutPath: testOut})
		},
		"ExportXML": func(a *App) (any, error) {
			return a.ExportXML(context.Background(), desktopapi.ExportXMLInput{CNPJ: testCNPJ, ChaveAcesso: testChave, OutPath: testOut})
		},
		"FetchNFeByChave": func(a *App) (any, error) {
			return a.FetchNFeByChave(context.Background(), desktopapi.FetchByChaveInput{CNPJ: testCNPJ, ChaveAcesso: testChave})
		},
		"ImportCTeXML": func(a *App) (any, error) {
			return a.ImportCTeXML(context.Background(), desktopapi.ImportXMLInput{CNPJ: testCNPJ, Files: []string{testXML}})
		},
		"ImportNFeXML": func(a *App) (any, error) {
			return a.ImportNFeXML(context.Background(), desktopapi.ImportXMLInput{CNPJ: testCNPJ, Files: []string{testXML}})
		},
		"ListCTe":         func(a *App) (any, error) { return a.ListCTe(desktopapi.ListCTeInput{CNPJ: testCNPJ}) },
		"ListCTeEvents":   func(a *App) (any, error) { return a.ListCTeEvents(desktopapi.CTeKeyInput{CNPJ: testCNPJ}) },
		"ListCompanies":   func(a *App) (any, error) { return a.ListCompanies() },
		"ListCredentials": func(a *App) (any, error) { return a.ListCredentials() },
		"ListDocuments":   func(a *App) (any, error) { return a.ListDocuments(desktopapi.ListInput{CNPJ: testCNPJ}) },
		"ListEventsForDocument": func(a *App) (any, error) {
			return a.ListEventsForDocument("doc-1")
		},
		"ListNFe":       func(a *App) (any, error) { return a.ListNFe(desktopapi.ListNFeInput{CNPJ: testCNPJ}) },
		"ListNFeEvents": func(a *App) (any, error) { return a.ListNFeEvents(desktopapi.NFeKeyInput{CNPJ: testCNPJ}) },
		"ListNFePendingManifestacoes": func(a *App) (any, error) {
			return a.ListNFePendingManifestacoes(desktopapi.NFePendingInput{CNPJ: testCNPJ})
		},
		"MarkCTeViewed": func(a *App) (any, error) { return a.MarkCTeViewed(desktopapi.MarkViewedInput{CNPJ: testCNPJ}) },
		"MarkDocumentsViewed": func(a *App) (any, error) {
			return a.MarkDocumentsViewed(desktopapi.MarkViewedInput{CNPJ: testCNPJ})
		},
		"MarkNFeViewed": func(a *App) (any, error) { return a.MarkNFeViewed(desktopapi.MarkViewedInput{CNPJ: testCNPJ}) },
		"PlanNFeCiencia": func(a *App) (any, error) {
			return a.PlanNFeCiencia(desktopapi.RegisterNFeCienciaInput{CNPJ: testCNPJ})
		},
		"PreviewResetCTe": func(a *App) (any, error) { return a.PreviewResetCTe(testCNPJ) },
		"Pull":            func(a *App) (any, error) { return a.Pull(context.Background(), desktopapi.PullInput{CNPJ: testCNPJ}) },
		"PullCTe": func(a *App) (any, error) {
			return a.PullCTe(context.Background(), desktopapi.PullCTeInput{CNPJ: testCNPJ})
		},
		"PullNFe": func(a *App) (any, error) {
			return a.PullNFe(context.Background(), desktopapi.PullNFeInput{CNPJ: testCNPJ})
		},
		"QueryNFSeEvents": func(a *App) (any, error) {
			return a.QueryNFSeEvents(context.Background(), desktopapi.QueryNFSeInput{CompanyCNPJ: testCNPJ, ChaveAcesso: testChave})
		},
		"RegisterNFeCiencia": func(a *App) (any, error) {
			return a.RegisterNFeCiencia(desktopapi.RegisterNFeCienciaInput{CNPJ: testCNPJ})
		},
		"RegisterNFeManifestacao": func(a *App) (any, error) {
			return a.RegisterNFeManifestacao(desktopapi.RegisterNFeManifestacaoInput{CNPJ: testCNPJ})
		},
		"ResetCTe": func(a *App) (any, error) { return a.ResetCTe(testCNPJ) },
		"ResetNFe": func(a *App) (any, error) { return a.ResetNFe(testCNPJ) },
		"ResetSyncState": func(a *App) (any, error) {
			return nil, a.ResetSyncState(desktopapi.ResetSyncInput{CompanyCNPJ: testCNPJ})
		},
		"Status":            func(a *App) (any, error) { return a.Status(testCNPJ) },
		"StatusCTe":         func(a *App) (any, error) { return a.StatusCTe(testCNPJ) },
		"StatusNFe":         func(a *App) (any, error) { return a.StatusNFe(testCNPJ) },
		"TestCTeConnection": func(a *App) (any, error) { return a.TestCTeConnection(context.Background(), testCNPJ) },
		"TestConnection":    func(a *App) (any, error) { return a.TestConnection(context.Background(), testCNPJ) },
		"UpdateCompany": func(a *App) (any, error) {
			return nil, a.UpdateCompany(desktopapi.UpdateCompanyInput{CNPJ: testCNPJ, Environment: "producao"})
		},
		"UpdateCredentialData": func(a *App) (any, error) {
			return nil, a.UpdateCredentialData(desktopapi.UpdateCredentialDataInput{CredentialID: "cred-1"})
		},
		"UpdateCredentialPath": func(a *App) (any, error) {
			return nil, a.UpdateCredentialPath(desktopapi.UpdateCredentialPathInput{CredentialID: "cred-1"})
		},
	}
}

// withoutCore lists the bound methods that work without the core services:
// password prompts, dialogs, log settings and build or path information.
var withoutCore = map[string]bool{
	"CancelCertPassword": true,
	"ConfirmQuit":        true,
	"ExportLogs":         true,
	"GetBuildInfo":       true,
	"GetDataDirectory":   true,
	"OpenDataDirectory":  true,
	"OpenLogsDirectory":  true,
	"SelectCertificate":  true,
	"SelectSaveFile":     true,
	"SelectXMLFiles":     true,
	"SetLogLevel":        true,
	"SetWindowTheme":     true,
	"ShowInFolder":       true,
	"SubmitCertPassword": true,
}

// serviceLifecycle lists the Wails service hooks. Wails calls them and does
// not bind them for the frontend.
var serviceLifecycle = map[string]bool{
	"ServiceStartup":  true,
	"ServiceShutdown": true,
}

// TestCoreRunsCoverEveryBoundMethod keeps coreRuns in step with App: a new
// bound method either needs the core, and goes in coreRuns, or is listed in
// withoutCore.
func TestCoreRunsCoverEveryBoundMethod(t *testing.T) {
	runs := coreRuns()
	var missing []string
	appType := reflect.TypeFor[*App]()
	for i := range appType.NumMethod() {
		name := appType.Method(i).Name
		if _, ok := runs[name]; !ok && !withoutCore[name] && !serviceLifecycle[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("bound methods in neither coreRuns nor withoutCore: %s", strings.Join(missing, ", "))
	}
}

func TestCoreMethodsRefuseCallsWithoutStartup(t *testing.T) {
	cause := errors.New("abrir banco de dados: disco cheio")
	tests := []struct {
		name       string
		startupErr error
		wantText   string
	}{
		{"startup failed", cause, "o nanci não iniciou: abrir banco de dados: disco cheio"},
		{"startup never ran", nil, "o nanci não iniciou"},
	}
	for _, tt := range tests {
		for name, run := range coreRuns() {
			t.Run(tt.name+"/"+name, func(t *testing.T) {
				a := newTestApp(services{})
				a.startupErr = tt.startupErr

				got, err := run(a)
				if !errors.Is(err, errNotStarted) {
					t.Fatalf("err = %v, want %v", err, errNotStarted)
				}
				if err.Error() != tt.wantText {
					t.Errorf("err = %q, want %q", err, tt.wantText)
				}
				if tt.startupErr != nil && !errors.Is(err, tt.startupErr) {
					t.Errorf("err = %v, want it to wrap %v", err, tt.startupErr)
				}
				if got != nil && !reflect.ValueOf(got).IsZero() {
					t.Errorf("result = %+v, want zero", got)
				}
			})
		}
	}
}

package desktopapi

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/vasfvitor/nanci/internal/app"
)

func TestImportXMLResultFrom(t *testing.T) {
	got := ImportXMLResultFrom(app.ImportXMLResult{
		Files: []app.ImportedFile{
			{Path: "a.xml", Kind: "resumo", ChaveAcesso: "35260912345678000195550010000123451000123456", Outcome: "imported"},
			{Path: "b.xml", Outcome: "rejected", Message: "XML ilegível"},
		},
		Imported: 1,
		Rejected: 1,
	})
	want := ImportXMLResult{
		Files: []ImportedFile{
			{Path: "a.xml", Kind: "resumo", ChaveAcesso: "35260912345678000195550010000123451000123456", Outcome: "imported"},
			{Path: "b.xml", Outcome: "rejected", Message: "XML ilegível"},
		},
		Imported: 1,
		Rejected: 1,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ImportXMLResultFrom = %+v, want %+v", got, want)
	}
}

func TestImportXMLResultFromEmptyFilesIsNotNil(t *testing.T) {
	got := ImportXMLResultFrom(app.ImportXMLResult{})
	if got.Files == nil || len(got.Files) != 0 {
		t.Errorf("Files = %v, want an empty slice", got.Files)
	}
}

func TestFetchByChaveResultFrom(t *testing.T) {
	got := FetchByChaveResultFrom(app.FetchByChaveResult{
		ChaveAcesso:      "35260912345678000195550010000123451000123456",
		Outcome:          "not_found",
		Message:          "A SEFAZ não encontrou a NF-e.",
		RequestsLastHour: 5,
		RequestBudget:    20,
	})
	want := FetchByChaveResult{
		ChaveAcesso:      "35260912345678000195550010000123451000123456",
		Outcome:          "not_found",
		Message:          "A SEFAZ não encontrou a NF-e.",
		RequestsLastHour: 5,
		RequestBudget:    20,
	}
	if got != want {
		t.Errorf("FetchByChaveResultFrom = %+v, want %+v", got, want)
	}
}

// TestBuildInfoJSON pins the keys the Sobre section reads.
func TestBuildInfoJSON(t *testing.T) {
	got, err := json.Marshal(BuildInfo{Version: "v1.2.3", Commit: "abc1234", Date: "2026-10-17", WebView2Version: "141.0.3537.71"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"version":"v1.2.3","commit":"abc1234","date":"2026-10-17","webview2Version":"141.0.3537.71"}`
	if string(got) != want {
		t.Errorf("BuildInfo JSON = %s, want %s", got, want)
	}
}

func TestCertPasswordRequestFrom(t *testing.T) {
	req := app.CertPasswordRequest{ //nolint:gosec // G101 reads CredentialID as a secret; it is a fixture ID
		RequestID:       "req-1",
		CompanyID:       "company-1",
		CompanyName:     "Empresa Teste",
		TargetCNPJ:      "12345678000195",
		CredentialID:    "credential-1",
		CredentialLabel: "Certificado A1",
		CertPath:        `C:\certs\a1.pfx`,
		Purpose:         "Sincronização NFS-e",
	}
	want := CertPasswordRequest{ //nolint:gosec // G101 reads CredentialID as a secret; it is a fixture ID
		RequestID:       "req-1",
		CompanyID:       "company-1",
		CompanyName:     "Empresa Teste",
		TargetCNPJ:      "12345678000195",
		CredentialID:    "credential-1",
		CredentialLabel: "Certificado A1",
		CertPath:        `C:\certs\a1.pfx`,
		Purpose:         "Sincronização NFS-e",
	}
	if got := CertPasswordRequestFrom(req); got != want {
		t.Errorf("CertPasswordRequestFrom = %+v, want %+v", got, want)
	}
}

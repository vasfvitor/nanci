package main

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/desktop/desktopapi"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

func date(t *testing.T, s string) *time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatal(err)
	}
	return &d
}

func TestAddCompanyMapsInput(t *testing.T) {
	tests := []struct {
		name       string
		policy     string
		date       string
		wantPolicy syncstate.SyncStartPolicy
		wantDate   string
	}{
		{"all", "all", "", syncstate.SyncStartPolicyAll, ""},
		{"since date", "since_date", "2026-01-15", syncstate.SyncStartPolicySinceDate, "2026-01-15"},
		{"from now with date", "from_now", "2026-03-01", syncstate.SyncStartPolicyFromNow, "2026-03-01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, a := newFakeApp(nil)

			err := a.AddCompany(desktopapi.AddCompanyInput{
				CNPJ:            testCNPJ,
				Name:            "Empresa Teste",
				CredentialID:    "cred-1",
				CredentialLabel: "Certificado",
				CertPath:        "C:/certs/empresa.pfx",
				Environment:     "producao_restrita",
				UF:              "SP",
				SyncStartPolicy: tt.policy,
				SyncStartDate:   tt.date,
			})
			if err != nil {
				t.Fatalf("AddCompany: %v", err)
			}

			want := company.AddCompanyInput{
				CNPJ:            testCNPJ,
				Name:            "Empresa Teste",
				CredentialID:    "cred-1",
				CredentialLabel: "Certificado",
				CertPath:        "C:/certs/empresa.pfx",
				Environment:     dfe.EnvironmentRestricted,
				UF:              "SP",
				SyncStartPolicy: tt.wantPolicy,
			}
			if tt.wantDate != "" {
				want.SyncStartDate = date(t, tt.wantDate)
			}
			assertCalls(t, &f.companies.recorder, call{"AddCompany", []any{want}})
		})
	}
}

// An empty policy means from_now starting today.
func TestAddCompanyDefaultsToFromNow(t *testing.T) {
	f, a := newFakeApp(nil)

	if err := a.AddCompany(desktopapi.AddCompanyInput{CNPJ: testCNPJ, Environment: "producao"}); err != nil {
		t.Fatalf("AddCompany: %v", err)
	}

	if len(f.companies.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(f.companies.calls))
	}
	got, ok := f.companies.calls[0].args[0].(company.AddCompanyInput)
	if !ok {
		t.Fatalf("argument is %T", f.companies.calls[0].args[0])
	}
	if got.SyncStartPolicy != syncstate.SyncStartPolicyFromNow || got.Environment != dfe.EnvironmentProduction {
		t.Errorf("policy, environment = %q, %q; want from_now, producao", got.SyncStartPolicy, got.Environment)
	}
	today := time.Now().Format("2006-01-02")
	if got.SyncStartDate == nil || got.SyncStartDate.Format("2006-01-02") != today {
		t.Errorf("SyncStartDate = %v, want %s", got.SyncStartDate, today)
	}
}

func TestAddCompanyRejectsBeforeCore(t *testing.T) {
	tests := []struct {
		name  string
		input desktopapi.AddCompanyInput
	}{
		{"unknown environment", desktopapi.AddCompanyInput{CNPJ: testCNPJ, Environment: "homologacao", SyncStartPolicy: "all"}},
		{"empty environment", desktopapi.AddCompanyInput{CNPJ: testCNPJ, SyncStartPolicy: "all"}},
		{"unknown policy", desktopapi.AddCompanyInput{CNPJ: testCNPJ, Environment: "producao", SyncStartPolicy: "sometimes"}},
		{"since date without date", desktopapi.AddCompanyInput{CNPJ: testCNPJ, Environment: "producao", SyncStartPolicy: "since_date"}},
		{"bad date", desktopapi.AddCompanyInput{CNPJ: testCNPJ, Environment: "producao", SyncStartPolicy: "since_date", SyncStartDate: "15/01/2026"}},
		{"all with date", desktopapi.AddCompanyInput{CNPJ: testCNPJ, Environment: "producao", SyncStartPolicy: "all", SyncStartDate: "2026-01-15"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, a := newFakeApp(nil)

			if err := a.AddCompany(tt.input); err == nil {
				t.Fatal("AddCompany succeeded, want an error")
			}
			assertCalls(t, &f.companies.recorder)
		})
	}
}

func TestUpdateCompanyMapsInput(t *testing.T) {
	f, a := newFakeApp(nil)

	err := a.UpdateCompany(desktopapi.UpdateCompanyInput{
		CNPJ:            testCNPJ,
		Name:            "Nome Novo",
		Environment:     "producao",
		UF:              "RJ",
		SyncStartPolicy: "since_date",
		SyncStartDate:   "2026-02-01",
	})
	if err != nil {
		t.Fatalf("UpdateCompany: %v", err)
	}

	assertCalls(t, &f.companies.recorder, call{"UpdateCompany", []any{company.UpdateCompanyInput{
		CNPJ:            testCNPJ,
		Name:            "Nome Novo",
		Environment:     dfe.EnvironmentProduction,
		UF:              "RJ",
		SyncStartPolicy: syncstate.SyncStartPolicySinceDate,
		SyncStartDate:   date(t, "2026-02-01"),
	}}})
}

// TestUpdateCompanyWithoutPolicy pins the current mapping of an empty policy:
// from_now with no date, unlike AddCompany, which also sets today's date.
func TestUpdateCompanyWithoutPolicy(t *testing.T) {
	f, a := newFakeApp(nil)

	if err := a.UpdateCompany(desktopapi.UpdateCompanyInput{CNPJ: testCNPJ, Name: "Nome", Environment: "producao"}); err != nil {
		t.Fatalf("UpdateCompany: %v", err)
	}

	assertCalls(t, &f.companies.recorder, call{"UpdateCompany", []any{company.UpdateCompanyInput{
		CNPJ:            testCNPJ,
		Name:            "Nome",
		Environment:     dfe.EnvironmentProduction,
		SyncStartPolicy: syncstate.SyncStartPolicyFromNow,
	}}})
}

func TestUpdateCompanyRejectsBeforeCore(t *testing.T) {
	tests := []struct {
		name  string
		input desktopapi.UpdateCompanyInput
	}{
		{"unknown environment", desktopapi.UpdateCompanyInput{CNPJ: testCNPJ, Environment: "homologacao"}},
		{"unknown policy", desktopapi.UpdateCompanyInput{CNPJ: testCNPJ, Environment: "producao", SyncStartPolicy: "sometimes"}},
		{"since date without date", desktopapi.UpdateCompanyInput{CNPJ: testCNPJ, Environment: "producao", SyncStartPolicy: "since_date"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, a := newFakeApp(nil)

			if err := a.UpdateCompany(tt.input); err == nil {
				t.Fatal("UpdateCompany succeeded, want an error")
			}
			assertCalls(t, &f.companies.recorder)
		})
	}
}

func TestCompanyAndCredentialWritesMapInput(t *testing.T) {
	tests := []struct {
		name string
		run  func(a *App) error
		want call
		// credential is true when the call goes to the credential service.
		credential bool
	}{
		{
			name: "AssignCredentialToCompany",
			run: func(a *App) error {
				return a.AssignCredentialToCompany(desktopapi.AssignCredentialInput{CompanyCNPJ: testCNPJ, CredentialID: "cred-2"})
			},
			want: call{"AssignCredentialToCompany", []any{company.AssignCredentialInput{CompanyCNPJ: testCNPJ, CredentialID: "cred-2"}}},
		},
		{
			name: "AddCredential",
			run: func(a *App) error {
				return a.AddCredential(desktopapi.AddCredentialInput{Label: "Certificado A1", CertPath: "C:/certs/a1.pfx"})
			},
			want:       call{"AddCredential", []any{credential.AddCredentialInput{Label: "Certificado A1", CertPath: "C:/certs/a1.pfx"}}},
			credential: true,
		},
		{
			name: "UpdateCredentialPath",
			run: func(a *App) error {
				return a.UpdateCredentialPath(desktopapi.UpdateCredentialPathInput{CredentialID: "cred-1", CertPath: "D:/novo.pfx"})
			},
			want:       call{"UpdateCredentialPath", []any{credential.UpdateCredentialPathInput{CredentialID: "cred-1", CertPath: "D:/novo.pfx"}}},
			credential: true,
		},
		{
			name: "UpdateCredentialData",
			run: func(a *App) error {
				return a.UpdateCredentialData(desktopapi.UpdateCredentialDataInput{CredentialID: "cred-1", Label: "Novo rótulo"})
			},
			want:       call{"UpdateCredentialData", []any{credential.UpdateCredentialDataInput{CredentialID: "cred-1", Label: "Novo rótulo"}}},
			credential: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, a := newFakeApp(nil)

			if err := tt.run(a); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			if tt.credential {
				assertCalls(t, &f.credentials.recorder, tt.want)
				assertCalls(t, &f.companies.recorder)
			} else {
				assertCalls(t, &f.companies.recorder, tt.want)
				assertCalls(t, &f.credentials.recorder)
			}
		})
	}
}

func TestCompanyAndCredentialMethodsPassCoreErrors(t *testing.T) {
	runs := map[string]func(a *App) (any, error){
		"AddCompany": func(a *App) (any, error) {
			return nil, a.AddCompany(desktopapi.AddCompanyInput{CNPJ: testCNPJ, Environment: "producao", SyncStartPolicy: "all"})
		},
		"UpdateCompany": func(a *App) (any, error) {
			return nil, a.UpdateCompany(desktopapi.UpdateCompanyInput{CNPJ: testCNPJ, Environment: "producao"})
		},
		"AssignCredentialToCompany": func(a *App) (any, error) {
			return nil, a.AssignCredentialToCompany(desktopapi.AssignCredentialInput{CompanyCNPJ: testCNPJ, CredentialID: "cred-1"})
		},
		"ListCompanies": func(a *App) (any, error) { return a.ListCompanies() },
		"AddCredential": func(a *App) (any, error) {
			return nil, a.AddCredential(desktopapi.AddCredentialInput{Label: "x", CertPath: "x.pfx"})
		},
		"ListCredentials": func(a *App) (any, error) { return a.ListCredentials() },
		"UpdateCredentialPath": func(a *App) (any, error) {
			return nil, a.UpdateCredentialPath(desktopapi.UpdateCredentialPathInput{CredentialID: "cred-1"})
		},
		"UpdateCredentialData": func(a *App) (any, error) {
			return nil, a.UpdateCredentialData(desktopapi.UpdateCredentialDataInput{CredentialID: "cred-1"})
		},
	}
	for name, run := range runs {
		t.Run(name, func(t *testing.T) {
			_, a := newFakeApp(errCore)

			got, err := run(a)
			if !errors.Is(err, errCore) {
				t.Fatalf("err = %v, want %v", err, errCore)
			}
			if got != nil && !reflect.ValueOf(got).IsNil() {
				t.Errorf("result = %#v, want nil", got)
			}
		})
	}
}

func TestListCompanies(t *testing.T) {
	f, a := newFakeApp(nil)
	f.companies.companies = []company.Company{{
		ID:              "company-1",
		CNPJ:            testCNPJ,
		Name:            "Empresa Teste",
		CredentialID:    "cred-1",
		Environment:     dfe.EnvironmentProduction,
		UF:              "SP",
		SyncStartPolicy: syncstate.SyncStartPolicyAll,
	}}

	got, err := a.ListCompanies()
	if err != nil {
		t.Fatalf("ListCompanies: %v", err)
	}

	assertCalls(t, &f.companies.recorder, call{"ListCompanies", nil})
	want := []desktopapi.CompanySummary{{
		ID:              "company-1",
		CNPJ:            testCNPJ,
		Name:            "Empresa Teste",
		CredentialID:    "cred-1",
		Environment:     "producao",
		UF:              "SP",
		SyncStartPolicy: "all",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("summaries = %+v, want %+v", got, want)
	}
}

func TestListCredentials(t *testing.T) {
	f, a := newFakeApp(nil)
	f.credentials.credentials = []credential.Credential{{
		ID:        "cred-1",
		Label:     "Certificado A1",
		CertPath:  "C:/certs/a1.pfx",
		OwnerCNPJ: testCNPJ,
	}}

	got, err := a.ListCredentials()
	if err != nil {
		t.Fatalf("ListCredentials: %v", err)
	}

	assertCalls(t, &f.credentials.recorder, call{"ListCredentials", nil})
	want := []desktopapi.CredentialSummary{{
		ID:        "cred-1",
		Label:     "Certificado A1",
		CertPath:  "C:/certs/a1.pfx",
		OwnerCNPJ: testCNPJ,
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("summaries = %+v, want %+v", got, want)
	}
}

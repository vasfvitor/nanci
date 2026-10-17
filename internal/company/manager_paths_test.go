package company_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

const (
	testCNPJ     = "11222333000181" // root 11222333
	testCNPJRoot = "11222333"
)

var errDB = errors.New("database is locked")

// stubStore holds at most one company and returns the configured errors.
type stubStore struct {
	company   *company.Company
	lookupErr error
	createErr error
	listErr   error
	updateErr error
	assignErr error

	created  []company.Company
	assigned dfe.CredentialID
}

func (s *stubStore) CreateCompany(_ context.Context, c *company.Company) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.created = append(s.created, *c)
	return nil
}

func (s *stubStore) ListCompanies(context.Context) ([]company.Company, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.company == nil {
		return nil, nil
	}
	return []company.Company{*s.company}, nil
}

func (s *stubStore) CompanyByCNPJ(context.Context, string) (*company.Company, error) {
	if s.lookupErr != nil {
		return nil, s.lookupErr
	}
	if s.company == nil {
		return nil, company.ErrCompanyNotFound
	}
	c := *s.company
	return &c, nil
}

func (s *stubStore) UpdateCompany(_ context.Context, c *company.Company) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	*s.company = *c
	return nil
}

func (s *stubStore) AssignCredential(_ context.Context, _ dfe.CompanyID, credentialID dfe.CredentialID) error {
	if s.assignErr != nil {
		return s.assignErr
	}
	s.assigned = credentialID
	return nil
}

// stubCreds knows the credentials in byID and records created ones.
type stubCreds struct {
	byID      map[dfe.CredentialID]*credential.Credential
	lookupErr error
	createErr error
	created   []credential.Credential
}

func (s *stubCreds) CredentialByID(_ context.Context, id dfe.CredentialID) (*credential.Credential, error) {
	if s.lookupErr != nil {
		return nil, s.lookupErr
	}
	cred, ok := s.byID[id]
	if !ok {
		return nil, credential.ErrCredentialNotFound
	}
	return cred, nil
}

func (s *stubCreds) CreateCredential(_ context.Context, cred *credential.Credential) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.created = append(s.created, *cred)
	return nil
}

type stubSyncs struct {
	snapshot    syncstate.SyncSnapshot
	snapshotErr error
	hasState    bool
	hasStateErr error
}

func (s *stubSyncs) LatestSyncSnapshot(context.Context, dfe.CompanyID, syncstate.SyncSource, dfe.Environment, string) (syncstate.SyncSnapshot, error) {
	return s.snapshot, s.snapshotErr
}

func (s *stubSyncs) HasSyncState(context.Context, syncstate.HasSyncStateParams) (bool, error) {
	return s.hasState, s.hasStateErr
}

func storedCompany() *company.Company {
	return &company.Company{
		ID:              "comp-1",
		CNPJ:            testCNPJ,
		CNPJRoot:        testCNPJRoot,
		Name:            "Empresa",
		CredentialID:    "cred-old",
		Environment:     dfe.EnvironmentProduction,
		SyncStartPolicy: syncstate.SyncStartPolicyFromNow,
	}
}

func wantErrContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want one containing %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want one containing %q", err, want)
	}
}

func TestManager_AddCompanyErrors(t *testing.T) {
	ctx := context.Background()
	certPath := filepath.Join(t.TempDir(), "cert.pfx")
	if err := os.WriteFile(certPath, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		input company.AddCompanyInput
		store *stubStore
		creds *stubCreds
		want  string
	}{
		{
			name:  "invalid CNPJ",
			input: company.AddCompanyInput{CNPJ: "11.222.333/0001-82", CredentialID: "cred-1"},
			want:  "CNPJ",
		},
		{
			name:  "unknown credential",
			input: company.AddCompanyInput{CNPJ: testCNPJ, CredentialID: "missing"},
			want:  "credencial não encontrada",
		},
		{
			name:  "credential lookup fails",
			input: company.AddCompanyInput{CNPJ: testCNPJ, CredentialID: "cred-1"},
			creds: &stubCreds{lookupErr: errDB},
			want:  "buscar credencial",
		},
		{
			name:  "missing certificate file",
			input: company.AddCompanyInput{CNPJ: testCNPJ, CertPath: filepath.Join(t.TempDir(), "none.pfx")},
			want:  "arquivo de certificado não encontrado",
		},
		{
			name:  "certificate path is a directory",
			input: company.AddCompanyInput{CNPJ: testCNPJ, CertPath: t.TempDir()},
			want:  "diretório",
		},
		{
			name:  "saving the new credential fails",
			input: company.AddCompanyInput{CNPJ: testCNPJ, CertPath: certPath},
			creds: &stubCreds{createErr: errDB},
			want:  "salvar credencial",
		},
		{
			name:  "saving the company fails",
			input: company.AddCompanyInput{CNPJ: testCNPJ, CredentialID: "cred-1"},
			store: &stubStore{createErr: errDB},
			want:  "salvar empresa",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.store
			if store == nil {
				store = &stubStore{}
			}
			creds := tt.creds
			if creds == nil {
				creds = &stubCreds{byID: map[dfe.CredentialID]*credential.Credential{"cred-1": {ID: "cred-1"}}}
			}
			m := company.NewManager(store, creds, &stubSyncs{})
			err := m.AddCompany(ctx, tt.input)
			wantErrContains(t, err, tt.want)
			if len(store.created) != 0 {
				t.Errorf("stored %d companies, want none", len(store.created))
			}
		})
	}
}

// TestManager_AddCompanyFromCertPath creates the credential from the
// certificate path; the label falls back to the company name, then the path.
func TestManager_AddCompanyFromCertPath(t *testing.T) {
	ctx := context.Background()
	certPath := filepath.Join(t.TempDir(), "cert.pfx")
	if err := os.WriteFile(certPath, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		label     string
		company   string
		wantLabel string
	}{
		{"explicit label", "Certificado A1", "Empresa", "Certificado A1"},
		{"company name", "", "Empresa", "Empresa"},
		{"certificate path", "", "", certPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &stubStore{}
			creds := &stubCreds{}
			m := company.NewManager(store, creds, &stubSyncs{})
			err := m.AddCompany(ctx, company.AddCompanyInput{
				CNPJ:            testCNPJ,
				Name:            tt.company,
				CredentialLabel: tt.label,
				CertPath:        certPath,
			})
			if err != nil {
				t.Fatalf("AddCompany: %v", err)
			}
			if len(creds.created) != 1 || len(store.created) != 1 {
				t.Fatalf("created %d credentials and %d companies, want 1 and 1", len(creds.created), len(store.created))
			}
			cred, comp := creds.created[0], store.created[0]
			if cred.Label != tt.wantLabel || cred.CertPath != certPath {
				t.Errorf("credential = (%q, %q), want (%q, %q)", cred.Label, cred.CertPath, tt.wantLabel, certPath)
			}
			if comp.CredentialID != cred.ID || comp.CredentialLabel != tt.wantLabel || comp.CNPJRoot != testCNPJRoot {
				t.Errorf("company = %+v, want credential %s and root %s", comp, cred.ID, testCNPJRoot)
			}
		})
	}
}

func TestManager_ListCompanies(t *testing.T) {
	ctx := context.Background()
	found := int64(42)
	success := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	finished := time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC)

	t.Run("snapshot fills the last sync fields", func(t *testing.T) {
		syncs := &stubSyncs{snapshot: syncstate.SyncSnapshot{
			State: &syncstate.SyncState{LastFoundNSU: &found, LastSuccessAt: &success},
			Run: &syncstate.SyncRun{
				Status:     syncstate.SyncStatusCompleted,
				StopReason: syncstate.SyncStopReasonCaughtUp,
				FinishedAt: &finished,
			},
		}}
		m := company.NewManager(&stubStore{company: storedCompany()}, &stubCreds{}, syncs)
		got, err := m.ListCompanies(ctx)
		if err != nil {
			t.Fatalf("ListCompanies: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d companies, want 1", len(got))
		}
		c := got[0]
		if c.LastFoundNSU == nil || *c.LastFoundNSU != found {
			t.Errorf("LastFoundNSU = %v, want %d", c.LastFoundNSU, found)
		}
		if c.LastSyncAt == nil || !c.LastSyncAt.Equal(finished) {
			t.Errorf("LastSyncAt = %v, want the run's FinishedAt %v", c.LastSyncAt, finished)
		}
		if c.LastRunStatus != syncstate.SyncStatusCompleted || c.LastRunStopReason != syncstate.SyncStopReasonCaughtUp {
			t.Errorf("last run = (%s, %s)", c.LastRunStatus, c.LastRunStopReason)
		}
	})

	t.Run("a running sync keeps the last success time", func(t *testing.T) {
		syncs := &stubSyncs{snapshot: syncstate.SyncSnapshot{
			State: &syncstate.SyncState{LastSuccessAt: &success},
			Run:   &syncstate.SyncRun{Status: syncstate.SyncStatusRunning},
		}}
		m := company.NewManager(&stubStore{company: storedCompany()}, &stubCreds{}, syncs)
		got, err := m.ListCompanies(ctx)
		if err != nil {
			t.Fatalf("ListCompanies: %v", err)
		}
		if got[0].LastSyncAt == nil || !got[0].LastSyncAt.Equal(success) {
			t.Errorf("LastSyncAt = %v, want %v", got[0].LastSyncAt, success)
		}
	})

	t.Run("store error", func(t *testing.T) {
		m := company.NewManager(&stubStore{listErr: errDB}, &stubCreds{}, &stubSyncs{})
		_, err := m.ListCompanies(ctx)
		wantErrContains(t, err, "listar empresas")
	})

	t.Run("snapshot error", func(t *testing.T) {
		m := company.NewManager(&stubStore{company: storedCompany()}, &stubCreds{}, &stubSyncs{snapshotErr: errDB})
		_, err := m.ListCompanies(ctx)
		wantErrContains(t, err, "carregar snapshot da empresa Empresa")
		if !errors.Is(err, errDB) {
			t.Errorf("error %v does not wrap the snapshot error", err)
		}
	})
}

func TestManager_AssignCredentialToCompany(t *testing.T) {
	ctx := context.Background()
	creds := map[dfe.CredentialID]*credential.Credential{
		"same-root":  {ID: "same-root", OwnerCNPJRoot: testCNPJRoot},
		"other-root": {ID: "other-root", OwnerCNPJRoot: "99888777"},
		"no-owner":   {ID: "no-owner"},
	}

	tests := []struct {
		name         string
		store        *stubStore
		credentialID string
		lookupErr    error
		wantErr      string
		wantIs       error
	}{
		{name: "same root", credentialID: "same-root"},
		{name: "credential without an owner root", credentialID: "no-owner"},
		{name: "other root", credentialID: "other-root", wantErr: "mesma raiz", wantIs: company.ErrCredentialMismatch},
		{name: "unknown credential", credentialID: "missing", wantErr: "credencial não encontrada"},
		{name: "credential lookup fails", credentialID: "same-root", lookupErr: errDB, wantErr: "buscar credencial", wantIs: errDB},
		{name: "unknown company", store: &stubStore{}, credentialID: "same-root", wantErr: "empresa não encontrada"},
		{
			name:         "store fails",
			store:        &stubStore{company: storedCompany(), assignErr: errDB},
			credentialID: "same-root",
			wantErr:      "atribuir credencial",
			wantIs:       errDB,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.store
			if store == nil {
				store = &stubStore{company: storedCompany()}
			}
			m := company.NewManager(store, &stubCreds{byID: creds, lookupErr: tt.lookupErr}, &stubSyncs{})
			err := m.AssignCredentialToCompany(ctx, company.AssignCredentialInput{
				CompanyCNPJ:  testCNPJ,
				CredentialID: tt.credentialID,
			})
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("AssignCredentialToCompany: %v", err)
				}
				if store.assigned != dfe.CredentialID(tt.credentialID) {
					t.Errorf("assigned %q, want %q", store.assigned, tt.credentialID)
				}
				return
			}
			wantErrContains(t, err, tt.wantErr)
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Errorf("error %v is not %v", err, tt.wantIs)
			}
			if store.assigned != "" {
				t.Errorf("assigned %q after an error", store.assigned)
			}
		})
	}
}

func TestManager_UpdateCompanyErrors(t *testing.T) {
	ctx := context.Background()
	date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		store   *stubStore
		syncs   *stubSyncs
		input   company.UpdateCompanyInput
		wantErr string
	}{
		{
			name:    "unknown company",
			store:   &stubStore{},
			input:   company.UpdateCompanyInput{CNPJ: testCNPJ},
			wantErr: "empresa não encontrada",
		},
		{
			name:    "sync state lookup fails",
			syncs:   &stubSyncs{hasStateErr: errDB},
			input:   company.UpdateCompanyInput{CNPJ: testCNPJ, SyncStartPolicy: syncstate.SyncStartPolicyAll},
			wantErr: "verificar estado de sincronização",
		},
		{
			name:    "a new start date is locked after the first sync",
			syncs:   &stubSyncs{hasState: true},
			input:   company.UpdateCompanyInput{CNPJ: testCNPJ, SyncStartPolicy: syncstate.SyncStartPolicyFromNow, SyncStartDate: &date},
			wantErr: "política inicial",
		},
		{
			name:    "store fails",
			store:   &stubStore{company: storedCompany(), updateErr: errDB},
			input:   company.UpdateCompanyInput{CNPJ: testCNPJ, Name: "Novo", SyncStartPolicy: syncstate.SyncStartPolicyFromNow},
			wantErr: "atualizar empresa",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.store
			if store == nil {
				store = &stubStore{company: storedCompany()}
			}
			syncs := tt.syncs
			if syncs == nil {
				syncs = &stubSyncs{}
			}
			m := company.NewManager(store, &stubCreds{}, syncs)
			err := m.UpdateCompany(ctx, tt.input)
			wantErrContains(t, err, tt.wantErr)
			if store.company != nil && store.company.Name != "Empresa" {
				t.Errorf("company renamed to %q after an error", store.company.Name)
			}
		})
	}
}

// TestManager_UpdateCompanySameStartDate: an unchanged policy and date do
// not consult the sync state, so a started sync does not block a rename.
func TestManager_UpdateCompanySameStartDate(t *testing.T) {
	ctx := context.Background()
	date := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	stored := storedCompany()
	stored.SyncStartPolicy = syncstate.SyncStartPolicySinceDate
	stored.SyncStartDate = &date
	store := &stubStore{company: stored}
	m := company.NewManager(store, &stubCreds{}, &stubSyncs{hasState: true, hasStateErr: errDB})

	sameDay := time.Date(2026, 3, 1, 15, 30, 0, 0, time.UTC)
	err := m.UpdateCompany(ctx, company.UpdateCompanyInput{
		CNPJ:            testCNPJ,
		Name:            "Renomeada",
		Environment:     dfe.EnvironmentProduction,
		SyncStartPolicy: syncstate.SyncStartPolicySinceDate,
		SyncStartDate:   &sameDay,
	})
	if err != nil {
		t.Fatalf("UpdateCompany: %v", err)
	}
	if store.company.Name != "Renomeada" {
		t.Errorf("Name = %q, want Renomeada", store.company.Name)
	}
}

func TestManager_CompanyByCNPJ(t *testing.T) {
	m := company.NewManager(&stubStore{company: storedCompany()}, &stubCreds{}, &stubSyncs{})
	got, err := m.CompanyByCNPJ(context.Background(), "11.222.333/0001-81")
	if err != nil {
		t.Fatalf("CompanyByCNPJ: %v", err)
	}
	if got.ID != "comp-1" {
		t.Errorf("ID = %q, want comp-1", got.ID)
	}
}

func TestLookupByCNPJ(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid CNPJ", func(t *testing.T) {
		_, err := company.LookupByCNPJ(ctx, &stubStore{company: storedCompany()}, "123")
		if err == nil {
			t.Fatal("error = nil, want a CNPJ validation error")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := company.LookupByCNPJ(ctx, &stubStore{}, testCNPJ)
		wantErrContains(t, err, "empresa não encontrada para o CNPJ 11.222.333/0001-81")
		if !errors.Is(err, company.ErrCompanyNotFound) {
			t.Errorf("error %v does not wrap ErrCompanyNotFound", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		_, err := company.LookupByCNPJ(ctx, &stubStore{lookupErr: errDB}, testCNPJ)
		wantErrContains(t, err, "buscar empresa")
		if !errors.Is(err, errDB) {
			t.Errorf("error %v does not wrap the database error", err)
		}
	})
}

func TestParseSyncStartPolicyInput(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	tests := []struct {
		name       string
		policy     string
		date       string
		wantPolicy syncstate.SyncStartPolicy
		wantDate   string // "" means a nil date
		wantErr    string
	}{
		{name: "empty defaults to from_now today", wantPolicy: syncstate.SyncStartPolicyFromNow, wantDate: today},
		{name: "from_now without a date", policy: "from_now", wantPolicy: syncstate.SyncStartPolicyFromNow, wantDate: today},
		{name: "from_now with a date", policy: "from_now", date: "2026-02-03", wantPolicy: syncstate.SyncStartPolicyFromNow, wantDate: "2026-02-03"},
		{name: "from_now with a bad date", policy: "from_now", date: "03/02/2026", wantErr: "YYYY-MM-DD"},
		{name: "all", policy: "all", wantPolicy: syncstate.SyncStartPolicyAll},
		{name: "all with a date", policy: "all", date: "2026-02-03", wantErr: "deve ficar vazio"},
		{name: "since_date", policy: "since_date", date: "2025-12-31", wantPolicy: syncstate.SyncStartPolicySinceDate, wantDate: "2025-12-31"},
		{name: "since_date without a date", policy: "since_date", wantErr: "obrigatório"},
		{name: "since_date with a bad date", policy: "since_date", date: "2025-13-01", wantErr: "YYYY-MM-DD"},
		{name: "unknown policy", policy: "sometimes", wantErr: "invalid sync start policy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy, date, err := company.ParseSyncStartPolicyInput(tt.policy, tt.date)
			if tt.wantErr != "" {
				wantErrContains(t, err, tt.wantErr)
				if policy != "" || date != nil {
					t.Errorf("got (%q, %v) with an error, want zero values", policy, date)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSyncStartPolicyInput: %v", err)
			}
			if policy != tt.wantPolicy {
				t.Errorf("policy = %q, want %q", policy, tt.wantPolicy)
			}
			switch {
			case tt.wantDate == "" && date != nil:
				t.Errorf("date = %v, want nil", date)
			case tt.wantDate != "" && (date == nil || date.Format("2006-01-02") != tt.wantDate):
				t.Errorf("date = %v, want %s", date, tt.wantDate)
			}
		})
	}
}

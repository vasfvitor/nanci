package company_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// Fake store implementing company.store interface
type fakeStore struct {
	companies []nfse.Company
}

func (f *fakeStore) CreateCompany(ctx context.Context, c *nfse.Company) error {
	f.companies = append(f.companies, *c)
	return nil
}
func (f *fakeStore) ListCompanies(ctx context.Context) ([]nfse.Company, error) {
	return f.companies, nil
}
func (f *fakeStore) CompanyByCNPJ(ctx context.Context, cnpj string) (*nfse.Company, error) {
	return &f.companies[0], nil
}
func (f *fakeStore) UpdateCompany(ctx context.Context, c *nfse.Company) error {
	f.companies[0] = *c
	return nil
}
func (f *fakeStore) AssignCredential(ctx context.Context, companyID dfe.CompanyID, credentialID nfse.CredentialID) error {
	return nil
}

type fakeCred struct{}

func (f *fakeCred) CredentialByID(ctx context.Context, id nfse.CredentialID) (*nfse.Credential, error) {
	return &nfse.Credential{ID: id}, nil
}
func (f *fakeCred) CreateCredential(ctx context.Context, cred *nfse.Credential) error { return nil }

// fakeSync reports a sync cursor for the sources in cursors; an empty
// source asks for any cursor.
type fakeSync struct {
	cursors []nfse.SyncSource
}

func (f *fakeSync) LatestSyncSnapshot(ctx context.Context, companyID dfe.CompanyID, source nfse.SyncSource, env nfse.Environment, cnpj string) (nfse.SyncSnapshot, error) {
	return nfse.SyncSnapshot{}, nil
}
func (f *fakeSync) HasSyncState(ctx context.Context, params nfse.HasSyncStateParams) (bool, error) {
	if params.Source == "" {
		return len(f.cursors) > 0, nil
	}
	return slices.Contains(f.cursors, params.Source), nil
}

func TestManager_AddCompany(t *testing.T) {
	s := &fakeStore{}
	m := company.NewManager(s, &fakeCred{}, &fakeSync{})

	err := m.AddCompany(context.Background(), company.AddCompanyInput{
		CNPJ:         "00.000.000/0001-91",
		Name:         "Test",
		Environment:  nfse.EnvironmentProduction,
		CredentialID: "cred-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.companies) != 1 {
		t.Fatalf("expected 1 company to be created")
	}
}

func TestManager_CompanyUF(t *testing.T) {
	ctx := context.Background()
	add := func(s *fakeStore, rawUF string) error {
		m := company.NewManager(s, &fakeCred{}, &fakeSync{})
		return m.AddCompany(ctx, company.AddCompanyInput{
			CNPJ:         "00.000.000/0001-91",
			Name:         "Test",
			Environment:  nfse.EnvironmentProduction,
			CredentialID: "cred-123",
			UF:           rawUF,
		})
	}

	for _, tt := range []struct{ raw, want string }{{"", ""}, {" sp ", "SP"}, {"DF", "DF"}} {
		s := &fakeStore{}
		if err := add(s, tt.raw); err != nil {
			t.Fatalf("AddCompany(UF %q): %v", tt.raw, err)
		}
		if got := s.companies[0].UF; got != tt.want {
			t.Errorf("AddCompany(UF %q) stored %q, want %q", tt.raw, got, tt.want)
		}
	}
	for _, rawUF := range []string{"XX", "São Paulo", "35"} {
		s := &fakeStore{}
		if err := add(s, rawUF); err == nil {
			t.Errorf("AddCompany(UF %q) succeeded, want error", rawUF)
		}
		if len(s.companies) != 0 {
			t.Errorf("AddCompany(UF %q) stored a company", rawUF)
		}
	}

	s := &fakeStore{}
	if err := add(s, "SP"); err != nil {
		t.Fatal(err)
	}
	m := company.NewManager(s, &fakeCred{}, &fakeSync{})
	stored := s.companies[0]
	update := company.UpdateCompanyInput{
		CNPJ:            stored.CNPJ,
		Name:            stored.Name,
		Environment:     stored.Environment,
		SyncStartPolicy: stored.SyncStartPolicy,
		SyncStartDate:   stored.SyncStartDate,
		UF:              "rj",
	}
	if err := m.UpdateCompany(ctx, update); err != nil {
		t.Fatalf("UpdateCompany: %v", err)
	}
	if got := s.companies[0].UF; got != "RJ" {
		t.Errorf("UF after update = %q, want RJ", got)
	}
	update.UF = "ZZ"
	if err := m.UpdateCompany(ctx, update); err == nil {
		t.Error("UpdateCompany with an invalid UF succeeded")
	}
	if got := s.companies[0].UF; got != "RJ" {
		t.Errorf("UF after rejected update = %q, want RJ", got)
	}
}

func TestManager_UpdateCompanySyncStartPolicyLock(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		cursors []nfse.SyncSource
		wantErr bool
	}{
		{"no cursor", nil, false},
		{"only an NF-e cursor", []nfse.SyncSource{nfse.SyncSourceNFe}, false},
		{"NFS-e cursor", []nfse.SyncSource{nfse.SyncSourceNFSe}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &fakeStore{}
			m := company.NewManager(s, &fakeCred{}, &fakeSync{cursors: tt.cursors})
			err := m.AddCompany(ctx, company.AddCompanyInput{
				CNPJ:            "00.000.000/0001-91",
				Name:            "Test",
				Environment:     nfse.EnvironmentProduction,
				CredentialID:    "cred-123",
				SyncStartPolicy: nfse.SyncStartPolicyFromNow,
			})
			if err != nil {
				t.Fatal(err)
			}
			stored := s.companies[0]
			err = m.UpdateCompany(ctx, company.UpdateCompanyInput{
				CNPJ:            stored.CNPJ,
				Name:            stored.Name,
				Environment:     stored.Environment,
				SyncStartPolicy: nfse.SyncStartPolicyAll,
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("UpdateCompany error = %v, wantErr %t", err, tt.wantErr)
			}
			want := nfse.SyncStartPolicyAll
			if tt.wantErr {
				want = nfse.SyncStartPolicyFromNow
			}
			if got := s.companies[0].SyncStartPolicy; got != want {
				t.Errorf("SyncStartPolicy = %s, want %s", got, want)
			}
		})
	}
}

func TestManager_UpdateCompanyEnvironmentLock(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		cursors []nfse.SyncSource
		wantErr bool
	}{
		{"no cursor", nil, false},
		{"only an NFS-e cursor", []nfse.SyncSource{nfse.SyncSourceNFSe}, false},
		{"NF-e cursor", []nfse.SyncSource{nfse.SyncSourceNFe}, true},
		{"NF-e and NFS-e cursors", []nfse.SyncSource{nfse.SyncSourceNFSe, nfse.SyncSourceNFe}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &fakeStore{}
			m := company.NewManager(s, &fakeCred{}, &fakeSync{cursors: tt.cursors})
			err := m.AddCompany(ctx, company.AddCompanyInput{
				CNPJ:            "00.000.000/0001-91",
				Name:            "Test",
				Environment:     nfse.EnvironmentRestricted,
				CredentialID:    "cred-123",
				SyncStartPolicy: nfse.SyncStartPolicyFromNow,
			})
			if err != nil {
				t.Fatal(err)
			}
			stored := s.companies[0]
			update := company.UpdateCompanyInput{
				CNPJ:            stored.CNPJ,
				Name:            "Renamed",
				Environment:     nfse.EnvironmentProduction,
				SyncStartPolicy: stored.SyncStartPolicy,
				SyncStartDate:   stored.SyncStartDate,
			}
			err = m.UpdateCompany(ctx, update)
			if tt.wantErr {
				if !errors.Is(err, company.ErrEnvironmentLocked) {
					t.Fatalf("UpdateCompany error = %v, want ErrEnvironmentLocked", err)
				}
				if got := s.companies[0]; got.Environment != nfse.EnvironmentRestricted || got.Name != "Test" {
					t.Errorf("company after refused update = (%s, %s), want (producao_restrita, Test)", got.Environment, got.Name)
				}

				// Other fields stay editable while the environment is kept.
				update.Environment = stored.Environment
				if err := m.UpdateCompany(ctx, update); err != nil {
					t.Fatalf("UpdateCompany keeping the environment: %v", err)
				}
				if got := s.companies[0].Name; got != "Renamed" {
					t.Errorf("Name = %s, want Renamed", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateCompany: %v", err)
			}
			if got := s.companies[0].Environment; got != nfse.EnvironmentProduction {
				t.Errorf("Environment = %s, want producao", got)
			}
		})
	}
}

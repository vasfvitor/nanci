package company_test

import (
	"context"
	"testing"

	"github.com/vasfvitor/nanci/internal/company"
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
func (f *fakeStore) UpdateCompany(ctx context.Context, c *nfse.Company) error { return nil }
func (f *fakeStore) AssignCredential(ctx context.Context, companyID nfse.CompanyID, credentialID nfse.CredentialID) error {
	return nil
}

type fakeCred struct{}

func (f *fakeCred) CredentialByID(ctx context.Context, id nfse.CredentialID) (*nfse.Credential, error) {
	return &nfse.Credential{ID: id}, nil
}
func (f *fakeCred) CreateCredential(ctx context.Context, cred *nfse.Credential) error { return nil }

type fakeSync struct{}

func (f *fakeSync) LatestSyncSnapshot(ctx context.Context, companyID nfse.CompanyID, env nfse.Environment, cnpj string) (nfse.SyncSnapshot, error) {
	return nfse.SyncSnapshot{}, nil
}
func (f *fakeSync) HasSyncState(ctx context.Context, params nfse.HasSyncStateParams) (bool, error) {
	return false, nil
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

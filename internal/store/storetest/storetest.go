package storetest

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
)

func OpenTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := store.OpenDB(context.Background(), filepath.Join(t.TempDir(), "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	return db
}

func TestCredential(id string) *nfse.Credential {
	return &nfse.Credential{
		ID:            nfse.CredentialID(id),
		Label:         "Certificate",
		CertPath:      `C:\certs\company.pfx`,
		OwnerCNPJ:     "11222333000181",
		OwnerCNPJRoot: "11222333",
	}
}

func TestCompany(id, cnpj string, env nfse.Environment, credential *nfse.Credential) *nfse.Company {
	return &nfse.Company{
		ID:                 nfse.CompanyID(id),
		CNPJ:               cnpj,
		CNPJRoot:           cnpj[:8],
		Name:               id,
		CredentialID:       credential.ID,
		CredentialLabel:    credential.Label,
		CredentialCertPath: credential.CertPath,
		Environment:        env,
	}
}

func Int64Ptr(v int64) *int64 { return &v }

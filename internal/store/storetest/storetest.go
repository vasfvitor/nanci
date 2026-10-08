package storetest

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/syncstate"
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

func TestCredential(id string) *credential.Credential {
	return &credential.Credential{
		ID:            dfe.CredentialID(id),
		Label:         "Certificate",
		CertPath:      `C:\certs\company.pfx`,
		OwnerCNPJ:     "11222333000181",
		OwnerCNPJRoot: "11222333",
	}
}

func TestCompany(id, cnpj string, env dfe.Environment, credential *credential.Credential) *company.Company {
	return &company.Company{
		ID:                 dfe.CompanyID(id),
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

// SetSyncStateLastSuccess stores the sync_state row of the company, source,
// environment and consultation CNPJ as if its last successful query was at,
// creating the row when missing.
func SetSyncStateLastSuccess(t *testing.T, db *sql.DB, companyID dfe.CompanyID, source syncstate.SyncSource, env dfe.Environment, cnpj string, at time.Time) {
	t.Helper()
	ts := at.UTC().Format(time.RFC3339)
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO sync_state (company_id, source, environment, consultation_cnpj, last_checked_nsu, last_success_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 10, ?, ?, ?)
		ON CONFLICT (company_id, source, environment, consultation_cnpj) DO UPDATE SET last_success_at = excluded.last_success_at
	`, string(companyID), string(source), string(env), cnpj, ts, ts, ts)
	if err != nil {
		t.Fatalf("set sync_state last success: %v", err)
	}
}

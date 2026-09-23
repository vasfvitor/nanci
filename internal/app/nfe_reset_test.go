package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sync"
)

func TestNFeResetUnlocksTheEnvironment(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedFixtures()
	ctx := context.Background()
	const now, blockedUntil = "2026-09-20T10:00:00Z", "2099-01-01T00:00:00Z"
	for _, stmt := range []string{
		`INSERT INTO sync_state (company_id, source, environment, consultation_cnpj, last_checked_nsu, created_at, updated_at)
		 VALUES ('comp-1', 'nfe', 'producao', '` + nfeTestCNPJ + `', 4, '` + now + `', '` + now + `')`,
		`INSERT INTO company_sync_sources (company_id, source, initial_sync_completed_at, blocked_until, blocked_reason, updated_at)
		 VALUES ('comp-1', 'nfe', '` + now + `', '` + blockedUntil + `', 'consumo_indevido', '` + now + `')`,
	} {
		if _, err := env.db.ExecContext(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}

	toRestricted := company.UpdateCompanyInput{
		CNPJ:            env.company.CNPJ,
		Name:            env.company.Name,
		Environment:     nfse.EnvironmentRestricted,
		UF:              env.company.UF,
		SyncStartPolicy: env.company.SyncStartPolicy,
		SyncStartDate:   env.company.SyncStartDate,
	}
	if err := env.app.Companies.UpdateCompany(ctx, toRestricted); !errors.Is(err, company.ErrEnvironmentLocked) {
		t.Fatalf("UpdateCompany before the reset = %v, want ErrEnvironmentLocked", err)
	}

	preview, err := env.app.NFe.PreviewReset(ctx, nfeTestCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if preview.CompanyDocuments != 3 || preview.Documents != 3 || preview.Events != 1 || preview.CompanyName != "Empresa Mock" {
		t.Errorf("PreviewReset = %+v, want 3 notes, 3 documents and 1 event", preview)
	}
	if docs, err := env.app.NFe.ListDocuments(ctx, NFeListInput{CNPJ: nfeTestCNPJ}); err != nil || len(docs) != 3 {
		t.Fatalf("documents after the preview = %d, %v; want 3", len(docs), err)
	}

	release, err := env.app.NFe.SyncManager.ReserveSource(env.company.ID, nfse.SyncSourceNFe)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.app.NFe.Reset(ctx, nfeTestCNPJ); !errors.Is(err, sync.ErrSyncRunning) {
		t.Errorf("Reset during a pull = %v, want ErrSyncRunning", err)
	}
	release()

	result, err := env.app.NFe.Reset(ctx, nfeTestCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if result.ResetCounts != preview.ResetCounts {
		t.Errorf("Reset = %+v, want the previewed %+v", result.ResetCounts, preview.ResetCounts)
	}
	if docs, err := env.app.NFe.ListDocuments(ctx, NFeListInput{CNPJ: nfeTestCNPJ}); err != nil || len(docs) != 0 {
		t.Errorf("documents after the reset = %d, %v; want none", len(docs), err)
	}
	var cursors int
	if err := env.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_state WHERE company_id = 'comp-1' AND source = 'nfe'`).Scan(&cursors); err != nil {
		t.Fatal(err)
	}
	var initialSync, blocked sql.NullString
	err = env.db.QueryRowContext(ctx, `SELECT initial_sync_completed_at, blocked_until FROM company_sync_sources WHERE company_id = 'comp-1' AND source = 'nfe'`).Scan(&initialSync, &blocked)
	if err != nil {
		t.Fatal(err)
	}
	if cursors != 0 || initialSync.Valid || blocked.String != blockedUntil {
		t.Errorf("after the reset: cursors %d, initial sync %v, blocked until %v; want 0, NULL and the SEFAZ block kept", cursors, initialSync, blocked)
	}

	if err := env.app.Companies.UpdateCompany(ctx, toRestricted); err != nil {
		t.Errorf("UpdateCompany after the reset: %v", err)
	}
}

package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store/storetest"
	"github.com/vasfvitor/nanci/internal/sync"
)

func TestCTeReset(t *testing.T) {
	env := newNFeTestEnv(t)
	ctx := context.Background()

	// A second company, the remetente of procte.xml, sees it too.
	other := storetest.TestCompany("comp-2", cteRemetenteCNPJ, nfse.EnvironmentProduction, &nfse.Credential{ID: "cred-1"})
	other.Name = "Remetente"
	if err := company.NewStore(env.db).CreateCompany(ctx, other); err != nil {
		t.Fatal(err)
	}
	env.seedCTe(env.company.ID, env.company.CNPJ, "procte.xml", 1)
	env.seedCTe(env.company.ID, env.company.CNPJ, "procte-toma4.xml", 2)
	env.seedCTe(env.company.ID, env.company.CNPJ, "procteos.xml", 3)
	env.seedCTe(env.company.ID, env.company.CNPJ, "proceventocte-cancelamento.xml", 4)
	env.seedCTe(env.company.ID, env.company.CNPJ, "proceventocte-comprovante.xml", 5)
	env.seedCTe(other.ID, other.CNPJ, "procte.xml", 1)

	const now, blockedUntil = "2026-09-20T10:00:00Z", "2099-01-01T00:00:00Z"
	for _, stmt := range []string{
		`INSERT INTO sync_state (company_id, source, environment, consultation_cnpj, last_checked_nsu, created_at, updated_at)
		 VALUES ('comp-1', 'cte', 'producao', '` + nfeTestCNPJ + `', 5, '` + now + `', '` + now + `')`,
		`INSERT INTO sync_state (company_id, source, environment, consultation_cnpj, last_checked_nsu, created_at, updated_at)
		 VALUES ('comp-1', 'nfe', 'producao', '` + nfeTestCNPJ + `', 7, '` + now + `', '` + now + `')`,
		`INSERT INTO company_sync_sources (company_id, source, initial_sync_completed_at, blocked_until, blocked_reason, updated_at)
		 VALUES ('comp-1', 'cte', '` + now + `', '` + blockedUntil + `', 'consumo_indevido', '` + now + `')`,
	} {
		if _, err := env.db.ExecContext(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}

	preview, err := env.app.CTe.PreviewReset(ctx, nfeTestCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	wantCounts := cte.ResetCounts{CompanyDocuments: 3, Documents: 2, Events: 1}
	if preview.ResetCounts != wantCounts || preview.CompanyName != "Empresa Mock" || preview.Environment != nfse.EnvironmentProduction {
		t.Errorf("PreviewReset = %+v, want %+v (procte stays for the remetente)", preview, wantCounts)
	}
	if docs, err := env.app.CTe.ListDocuments(ctx, ListCTeInput{CNPJ: nfeTestCNPJ}); err != nil || len(docs) != 3 {
		t.Fatalf("documents after the preview = %d, %v; want 3", len(docs), err)
	}

	release, err := env.app.CTe.SyncManager.ReserveSource(env.company.ID, nfse.SyncSourceCTe)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.app.CTe.Reset(ctx, nfeTestCNPJ); !errors.Is(err, sync.ErrSyncRunning) {
		t.Errorf("Reset during a pull = %v, want ErrSyncRunning", err)
	}
	release()

	// An NF-e pull does not hold the CT-e source.
	releaseNFe, err := env.app.CTe.SyncManager.ReserveSource(env.company.ID, nfse.SyncSourceNFe)
	if err != nil {
		t.Fatal(err)
	}
	result, err := env.app.CTe.Reset(ctx, nfeTestCNPJ)
	releaseNFe()
	if err != nil {
		t.Fatal(err)
	}
	if result.ResetCounts != preview.ResetCounts {
		t.Errorf("Reset = %+v, want the previewed %+v", result.ResetCounts, preview.ResetCounts)
	}
	if docs, err := env.app.CTe.ListDocuments(ctx, ListCTeInput{CNPJ: nfeTestCNPJ}); err != nil || len(docs) != 0 {
		t.Errorf("documents after the reset = %d, %v; want none", len(docs), err)
	}

	otherDocs, err := env.app.CTe.ListDocuments(ctx, ListCTeInput{CNPJ: cteRemetenteCNPJ})
	if err != nil {
		t.Fatal(err)
	}
	if len(otherDocs) != 1 || otherDocs[0].ChaveAcesso != cteChaveProc || otherDocs[0].CompanyRole != cte.CompanyRoleRemetente {
		t.Errorf("remetente documents = %+v, want procte.xml kept", otherDocs)
	}
	if events, err := env.app.CTe.ListEvents(ctx, cteRemetenteCNPJ, cteChaveProc); err != nil || len(events) != 1 {
		t.Errorf("remetente events = %d, %v; want the cancelamento kept", len(events), err)
	}

	cursors := func(source string) int {
		t.Helper()
		var n int
		if err := env.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_state WHERE company_id = 'comp-1' AND source = ?`, source).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	var initialSync, blocked sql.NullString
	err = env.db.QueryRowContext(ctx, `SELECT initial_sync_completed_at, blocked_until FROM company_sync_sources WHERE company_id = 'comp-1' AND source = 'cte'`).Scan(&initialSync, &blocked)
	if err != nil {
		t.Fatal(err)
	}
	if cursors("cte") != 0 || initialSync.Valid || blocked.String != blockedUntil {
		t.Errorf("after the reset: cte cursors %d, initial sync %v, blocked until %v; want 0, NULL and the SEFAZ block kept", cursors("cte"), initialSync, blocked)
	}
	if cursors("nfe") != 1 {
		t.Error("the CT-e reset removed the NF-e cursor")
	}
}

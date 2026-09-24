package app

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sync"
)

func TestNFeReset(t *testing.T) {
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

	// A homologação note left from an earlier environment goes too.
	env.seed("procnfe.xml", 5, nfeChaveProc, testChave(t, 50), "<tpAmb>1</tpAmb>", "<tpAmb>2</tpAmb>")

	preview, err := env.app.NFe.PreviewReset(ctx, nfeTestCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if preview.CompanyDocuments != 4 || preview.Documents != 4 || preview.Events != 1 || preview.CompanyName != "Empresa Mock" {
		t.Errorf("PreviewReset = %+v, want 4 notes, 4 documents and 1 event", preview)
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
	var notes, cursors int
	if err := env.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM company_nfe_documents WHERE company_id = 'comp-1'`).Scan(&notes); err != nil {
		t.Fatal(err)
	}
	if notes != 0 {
		t.Errorf("company notes after the reset = %d, want none of either environment", notes)
	}
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
}

// TestNFeEnvironmentSwitchHidesAndShowsNotes switches a company that
// already synced NF-e in produção to homologação and back: each environment
// lists, counts and exports only its own notes.
func TestNFeEnvironmentSwitchHidesAndShowsNotes(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedFixtures()
	ctx := context.Background()
	if _, err := env.db.ExecContext(ctx, `
		INSERT INTO sync_state (company_id, source, environment, consultation_cnpj, last_checked_nsu, created_at, updated_at)
		VALUES ('comp-1', 'nfe', 'producao', ?, 4, '2026-09-20T10:00:00Z', '2026-09-20T10:00:00Z')
	`, nfeTestCNPJ); err != nil {
		t.Fatal(err)
	}

	switchTo := func(environment nfse.Environment) {
		t.Helper()
		err := env.app.Companies.UpdateCompany(ctx, company.UpdateCompanyInput{
			CNPJ:            env.company.CNPJ,
			Name:            env.company.Name,
			Environment:     environment,
			UF:              env.company.UF,
			SyncStartPolicy: env.company.SyncStartPolicy,
			SyncStartDate:   env.company.SyncStartDate,
		})
		if err != nil {
			t.Fatalf("switch to %s: %v", environment, err)
		}
	}
	listed := func() []string {
		t.Helper()
		docs, err := env.app.NFe.ListDocuments(ctx, NFeListInput{CNPJ: nfeTestCNPJ})
		if err != nil {
			t.Fatal(err)
		}
		return chavesOf(docs)
	}
	status := func() NFeStatusResult {
		t.Helper()
		s, err := env.app.NFe.Status(ctx, nfeTestCNPJ)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	producao := []string{nfeChaveDenegada, nfeChaveProc, nfeChaveCancelada}

	switchTo(nfse.EnvironmentRestricted)
	if got := listed(); len(got) != 0 {
		t.Errorf("homologação lists %v, want none", got)
	}
	if s := status(); s.TpAmb != "2" || s.TotalDestinatario != 0 || s.TotalResumos+s.TotalCompletas != 0 || s.PendingCiencia+s.PendingConclusiva != 0 {
		t.Errorf("homologação status = %+v, want tpAmb 2 and no notes", s)
	}
	if pending, err := env.app.NFe.ListPendingManifestacoes(ctx, NFePendingInput{CNPJ: nfeTestCNPJ}); err != nil || len(pending) != 0 {
		t.Errorf("homologação pending = %d, %v; want none", len(pending), err)
	}
	if _, err := env.app.NFe.ListEvents(ctx, nfeTestCNPJ, nfeChaveProc); !errors.Is(err, nfe.ErrDocumentNotFound) {
		t.Errorf("ListEvents of a produção note in homologação = %v, want ErrDocumentNotFound", err)
	}
	export, err := env.app.NFe.ExportXMLZip(ctx, NFeExportInput{CNPJ: nfeTestCNPJ, IncludeResumos: true, OutPath: filepath.Join(t.TempDir(), "hom.zip")})
	if err != nil || export.ExportedCount != 0 {
		t.Errorf("homologação export = %d, %v; want nothing", export.ExportedCount, err)
	}

	homologacao := testChave(t, 50)
	env.seed("procnfe.xml", 5, nfeChaveProc, homologacao, "<tpAmb>1</tpAmb>", "<tpAmb>2</tpAmb>")
	if got := listed(); !slices.Equal(got, []string{homologacao}) {
		t.Errorf("homologação lists %v, want only its own note", got)
	}

	switchTo(nfse.EnvironmentProduction)
	if got := listed(); !slices.Equal(got, producao) {
		t.Errorf("back in produção lists %v, want %v", got, producao)
	}
	if s := status(); s.TpAmb != "1" || s.TotalResumos+s.TotalCompletas != 3 {
		t.Errorf("back in produção status = %+v, want the 3 produção notes", s)
	}
	if events, err := env.app.NFe.ListEvents(ctx, nfeTestCNPJ, nfeChaveProc); err != nil || len(events) != 1 {
		t.Errorf("produção events = %d, %v; want the ciência", len(events), err)
	}
}

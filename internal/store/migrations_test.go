package store_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/pressly/goose/v3"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/store"
)

func TestMigration007KeysSyncStateAndRunsBySource(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, filepath.Join(t.TempDir(), "migrate.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrations, err := store.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 6); err != nil {
		t.Fatalf("migrate to version 6: %v", err)
	}

	const now = "2026-06-01T10:00:00Z"
	mustExec(t, db, `
		INSERT INTO credentials (id, label, cert_path, owner_cnpj, owner_cnpj_root, fingerprint_sha256, subject_name, created_at, updated_at)
		VALUES ('cred-1', 'Certificate', 'company.pfx', '11222333000181', '11222333', 'fp', 'CN=Company', ?, ?)
	`, now, now)
	mustExec(t, db, `
		INSERT INTO companies (id, cnpj, cnpj_root, name, credential_id, environment, sync_start_policy, initial_sync_completed_at, created_at, updated_at)
		VALUES ('comp-1', '11222333000181', '11222333', 'Synced', 'cred-1', 'producao', 'all', ?, ?, ?)
	`, now, now, now)
	mustExec(t, db, `
		INSERT INTO companies (id, cnpj, cnpj_root, name, credential_id, environment, sync_start_policy, initial_sync_completed_at, created_at, updated_at)
		VALUES ('comp-2', '11222333000262', '11222333', 'Never synced', 'cred-1', 'producao', 'all', NULL, ?, ?)
	`, now, now)
	mustExec(t, db, `
		INSERT INTO sync_state (company_id, environment, consultation_cnpj, last_checked_nsu, last_found_nsu, last_empty_streak, created_at, updated_at)
		VALUES ('comp-1', 'producao', '11222333000181', 42, 40, 1, ?, ?)
	`, now, now)
	mustExec(t, db, `
		INSERT INTO sync_runs (id, company_id, credential_id, environment, credential_cnpj, consultation_cnpj, consultation_basis, mode, started_at, from_nsu, to_nsu, status)
		VALUES ('run-1', 'comp-1', 'cred-1', 'producao', '11222333000181', '11222333000181', 'exact_certificate_cnpj', 'normal', ?, 0, 42, 'running')
	`, now)

	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	var stateSource string
	var lastChecked int64
	var maxNSU sql.NullInt64
	if err := db.QueryRowContext(ctx, `SELECT source, last_checked_nsu, max_nsu FROM sync_state WHERE company_id = 'comp-1'`).
		Scan(&stateSource, &lastChecked, &maxNSU); err != nil {
		t.Fatalf("read migrated sync_state: %v", err)
	}
	if stateSource != "nfse" || lastChecked != 42 || maxNSU.Valid {
		t.Errorf("sync_state = (%q, %d, %v), want (nfse, 42, NULL)", stateSource, lastChecked, maxNSU)
	}

	var runSource string
	if err := db.QueryRowContext(ctx, `SELECT source FROM sync_runs WHERE id = 'run-1'`).Scan(&runSource); err != nil {
		t.Fatalf("read migrated sync_runs: %v", err)
	}
	if runSource != "nfse" {
		t.Errorf("sync_runs.source = %q, want nfse", runSource)
	}

	rows, err := db.QueryContext(ctx, `SELECT company_id, source, initial_sync_completed_at FROM company_sync_sources`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var backfilled []string
	for rows.Next() {
		var companyID, source string
		var completedAt sql.NullString
		if err := rows.Scan(&companyID, &source, &completedAt); err != nil {
			t.Fatal(err)
		}
		if source != "nfse" || completedAt.String != now {
			t.Errorf("company_sync_sources row for %s = (%q, %v), want (nfse, %s)", companyID, source, completedAt, now)
		}
		backfilled = append(backfilled, companyID)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(backfilled) != 1 || backfilled[0] != "comp-1" {
		t.Errorf("company_sync_sources companies = %v, want [comp-1]", backfilled)
	}

	insertRunning := `
		INSERT INTO sync_runs (id, company_id, source, credential_id, environment, credential_cnpj, consultation_cnpj, consultation_basis, mode, started_at, from_nsu, to_nsu, status)
		VALUES (?, 'comp-1', ?, 'cred-1', 'producao', '11222333000181', '11222333000181', 'exact_certificate_cnpj', 'normal', ?, 0, 0, 'running')
	`
	if _, err := db.ExecContext(ctx, insertRunning, "run-nfe", "nfe", now); err != nil {
		t.Errorf("running nfe run next to a running nfse run was rejected: %v", err)
	}
	if _, err := db.ExecContext(ctx, insertRunning, "run-nfse-2", "nfse", now); err == nil {
		t.Error("second running nfse run for the same company was accepted")
	}

	if _, err := provider.DownTo(ctx, 6); err != nil {
		t.Fatalf("migrate down to version 6: %v", err)
	}
	var stateRows int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_state WHERE company_id = 'comp-1' AND last_checked_nsu = 42`).Scan(&stateRows); err != nil {
		t.Fatalf("read sync_state after down: %v", err)
	}
	if stateRows != 1 {
		t.Errorf("sync_state rows after down = %d, want 1", stateRows)
	}
}

func TestMigration008AddsNFeTables(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, filepath.Join(t.TempDir(), "migrate.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrations, err := store.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 7); err != nil {
		t.Fatalf("migrate to version 7: %v", err)
	}

	nfeTables := []string{"nfe_documents", "company_nfe_documents", "nfe_events", "nfe_manifestations", "company_nfe_export_marks"}
	countTables := func() int {
		t.Helper()
		var n int
		err := db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM sqlite_master
			WHERE type = 'table' AND name IN (?, ?, ?, ?, ?)
		`, nfeTables[0], nfeTables[1], nfeTables[2], nfeTables[3], nfeTables[4]).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}

	if _, err := provider.UpTo(ctx, 8); err != nil {
		t.Fatalf("migrate to version 8: %v", err)
	}
	if n := countTables(); n != len(nfeTables) {
		t.Errorf("NF-e tables after up = %d, want %d", n, len(nfeTables))
	}

	const now = "2026-09-01T10:00:00Z"
	mustExec(t, db, `
		INSERT INTO companies (id, cnpj, cnpj_root, name, environment, sync_start_policy, created_at, updated_at)
		VALUES ('comp-1', '70860312000150', '70860312', 'Company', 'producao', 'all', ?, ?)
	`, now, now)
	mustExec(t, db, `
		INSERT INTO nfe_documents (id, chave_acesso, modelo, serie, numero, issue_date, competence, protocolo,
			emitente_cnpj, emitente_name, emitente_ie, emitente_uf, destinatario_cnpj, destinatario_name,
			transportador_cnpj, tp_nf, fin_nfe, nat_op, situacao, completeness, layout_version, raw_hash,
			created_at, updated_at)
		VALUES ('doc-1', '35260911222333000181550010000012341123456787', '55', '1', '1234', ?, '2026-09', '',
			'11222333000181', 'Emitente', '', 'SP', '', '', '', '1', '', '', 'autorizada', 'resumo', '1.01', 'hash',
			?, ?)
	`, now, now, now)
	mustExec(t, db, `
		INSERT INTO company_nfe_documents (relation_id, company_id, nfe_document_id, company_role, visibility_reason,
			first_synced_at, last_synced_at)
		VALUES ('rel-1', 'comp-1', 'doc-1', 'destinatario', 'resumo_destinatario', ?, ?)
	`, now, now)
	var manifestacao string
	if err := db.QueryRowContext(ctx, `SELECT manifestacao FROM company_nfe_documents WHERE relation_id = 'rel-1'`).Scan(&manifestacao); err != nil {
		t.Fatal(err)
	}
	if manifestacao != "nenhuma" {
		t.Errorf("default manifestacao = %q, want nenhuma", manifestacao)
	}

	if _, err := provider.DownTo(ctx, 7); err != nil {
		t.Fatalf("migrate down to version 7: %v", err)
	}
	if n := countTables(); n != 0 {
		t.Errorf("NF-e tables after down = %d, want 0", n)
	}
}

func TestMigration009AddsCompanyUF(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, filepath.Join(t.TempDir(), "migrate.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrations, err := store.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 8); err != nil {
		t.Fatalf("migrate to version 8: %v", err)
	}
	const now = "2026-09-01T10:00:00Z"
	mustExec(t, db, `
		INSERT INTO companies (id, cnpj, cnpj_root, name, environment, sync_start_policy, created_at, updated_at)
		VALUES ('comp-1', '70860312000150', '70860312', 'Company', 'producao', 'all', ?, ?)
	`, now, now)

	if _, err := provider.UpTo(ctx, 9); err != nil {
		t.Fatalf("migrate to version 9: %v", err)
	}
	var uf string
	if err := db.QueryRowContext(ctx, `SELECT uf FROM companies WHERE id = 'comp-1'`).Scan(&uf); err != nil {
		t.Fatalf("read companies.uf: %v", err)
	}
	if uf != "" {
		t.Errorf("uf of an existing company = %q, want empty", uf)
	}

	if _, err := provider.DownTo(ctx, 8); err != nil {
		t.Fatalf("migrate down to version 8: %v", err)
	}
	var companies int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM companies`).Scan(&companies); err != nil {
		t.Fatalf("read companies after down: %v", err)
	}
	if companies != 1 {
		t.Errorf("companies after down = %d, want 1", companies)
	}
}

func TestMigration010AddsSyncItemFailures(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, filepath.Join(t.TempDir(), "migrate.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrations, err := store.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 9); err != nil {
		t.Fatalf("migrate to version 9: %v", err)
	}
	const now = "2026-09-01T10:00:00Z"
	mustExec(t, db, `
		INSERT INTO companies (id, cnpj, cnpj_root, name, environment, sync_start_policy, created_at, updated_at)
		VALUES ('comp-1', '70860312000150', '70860312', 'Company', 'producao', 'all', ?, ?)
	`, now, now)
	mustExec(t, db, `
		INSERT INTO sync_state (company_id, source, environment, consultation_cnpj, last_checked_nsu, created_at, updated_at)
		VALUES ('comp-1', 'nfe', 'producao', '70860312000150', 42, ?, ?)
	`, now, now)

	if _, err := provider.UpTo(ctx, 10); err != nil {
		t.Fatalf("migrate to version 10: %v", err)
	}
	var failedNSU sql.NullInt64
	var attempts int
	if err := db.QueryRowContext(ctx, `SELECT failed_nsu, failed_nsu_attempts FROM sync_state WHERE company_id = 'comp-1'`).
		Scan(&failedNSU, &attempts); err != nil {
		t.Fatalf("read sync_state failure columns: %v", err)
	}
	if failedNSU.Valid || attempts != 0 {
		t.Errorf("failure columns of an existing state = (%v, %d), want (NULL, 0)", failedNSU, attempts)
	}

	if _, err := provider.DownTo(ctx, 9); err != nil {
		t.Fatalf("migrate down to version 9: %v", err)
	}
	var lastChecked int64
	if err := db.QueryRowContext(ctx, `SELECT last_checked_nsu FROM sync_state WHERE company_id = 'comp-1'`).Scan(&lastChecked); err != nil {
		t.Fatalf("read sync_state after down: %v", err)
	}
	if lastChecked != 42 {
		t.Errorf("last_checked_nsu after down = %d, want 42", lastChecked)
	}
	if _, err := db.ExecContext(ctx, `SELECT failed_nsu FROM sync_state`); err == nil {
		t.Error("failed_nsu still exists after down")
	}
}

func TestMigration011AddsManifestacaoTpAmb(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, filepath.Join(t.TempDir(), "migrate.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrations, err := store.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 10); err != nil {
		t.Fatalf("migrate to version 10: %v", err)
	}
	const now = "2026-09-01T10:00:00Z"
	mustExec(t, db, `
		INSERT INTO companies (id, cnpj, cnpj_root, name, environment, sync_start_policy, created_at, updated_at)
		VALUES ('comp-1', '70860312000150', '70860312', 'Company', 'producao', 'all', ?, ?)
	`, now, now)
	mustExec(t, db, `
		INSERT INTO nfe_manifestations (id, company_id, chave_acesso, tp_evento, n_seq_evento, justificativa,
			id_lote, status, c_stat, x_motivo, protocolo, created_at)
		VALUES ('m-1', 'comp-1', '35260911222333000181550010000012341123456787', '210210', 1, '',
			'1', 'registrada', '135', 'Evento registrado', '891260000000001', ?)
	`, now)

	if _, err := provider.UpTo(ctx, 11); err != nil {
		t.Fatalf("migrate to version 11: %v", err)
	}
	var tpAmb string
	if err := db.QueryRowContext(ctx, `SELECT tp_amb FROM nfe_manifestations WHERE id = 'm-1'`).Scan(&tpAmb); err != nil {
		t.Fatalf("read nfe_manifestations.tp_amb: %v", err)
	}
	if tpAmb != "" {
		t.Errorf("tp_amb of an existing manifestação = %q, want empty", tpAmb)
	}

	if _, err := provider.DownTo(ctx, 10); err != nil {
		t.Fatalf("migrate down to version 10: %v", err)
	}
	var rows int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nfe_manifestations`).Scan(&rows); err != nil {
		t.Fatalf("read nfe_manifestations after down: %v", err)
	}
	if rows != 1 {
		t.Errorf("nfe_manifestations rows after down = %d, want 1", rows)
	}
}

func TestMigration012IndexesCompanyNFeDocumentsByDocument(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, filepath.Join(t.TempDir(), "migrate.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrations, err := store.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 11); err != nil {
		t.Fatalf("migrate to version 11: %v", err)
	}
	countIndex := func() int {
		t.Helper()
		var n int
		err := db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM sqlite_master
			WHERE type = 'index' AND name = 'idx_company_nfe_documents_document'
		`).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}

	if _, err := provider.UpTo(ctx, 12); err != nil {
		t.Fatalf("migrate to version 12: %v", err)
	}
	if n := countIndex(); n != 1 {
		t.Errorf("idx_company_nfe_documents_document after up = %d, want 1", n)
	}

	if _, err := provider.DownTo(ctx, 11); err != nil {
		t.Fatalf("migrate down to version 11: %v", err)
	}
	if n := countIndex(); n != 0 {
		t.Errorf("idx_company_nfe_documents_document after down = %d, want 0", n)
	}
}

func TestMigration013RenamesManifestacoesAndChecksSyncRequestSource(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, filepath.Join(t.TempDir(), "migrate.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrations, err := store.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 12); err != nil {
		t.Fatalf("migrate to version 12: %v", err)
	}
	const now = "2026-09-01T10:00:00Z"
	mustExec(t, db, `
		INSERT INTO companies (id, cnpj, cnpj_root, name, environment, sync_start_policy, created_at, updated_at)
		VALUES ('comp-1', '70860312000150', '70860312', 'Company', 'producao', 'all', ?, ?)
	`, now, now)
	mustExec(t, db, `
		INSERT INTO nfe_manifestations (id, company_id, chave_acesso, tp_evento, n_seq_evento, justificativa,
			id_lote, status, c_stat, x_motivo, protocolo, created_at, tp_amb)
		VALUES ('m-1', 'comp-1', '35260911222333000181550010000012341123456787', '210210', 1, '',
			'1', 'registrada', '135', 'Evento registrado', '891260000000001', ?, '1')
	`, now)
	mustExec(t, db, `INSERT INTO sync_requests (company_id, source, requested_at) VALUES ('comp-1', 'nfe', ?)`, now)

	countIndex := func(name string) int {
		t.Helper()
		var n int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, name).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	countRows := func(table string) int {
		t.Helper()
		var n int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		return n
	}

	if _, err := provider.UpTo(ctx, 13); err != nil {
		t.Fatalf("migrate to version 13: %v", err)
	}
	if n := countRows("nfe_manifestacoes"); n != 1 {
		t.Errorf("nfe_manifestacoes rows = %d, want 1", n)
	}
	if n := countRows("sync_requests"); n != 1 {
		t.Errorf("sync_requests rows = %d, want 1", n)
	}
	for _, name := range []string{"idx_nfe_manifestacoes_company_chave", "idx_company_nfe_documents_viewed_at", "idx_sync_requests_window"} {
		if n := countIndex(name); n != 1 {
			t.Errorf("index %s after up = %d, want 1", name, n)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO sync_requests (company_id, source, requested_at) VALUES ('comp-1', 'bogus', ?)`, now); err == nil {
		t.Error("sync_requests accepted source 'bogus' after up, want a CHECK failure")
	}

	if _, err := provider.DownTo(ctx, 12); err != nil {
		t.Fatalf("migrate down to version 12: %v", err)
	}
	if n := countRows("nfe_manifestations"); n != 1 {
		t.Errorf("nfe_manifestations rows after down = %d, want 1", n)
	}
	if n := countRows("sync_requests"); n != 1 {
		t.Errorf("sync_requests rows after down = %d, want 1", n)
	}
	for _, name := range []string{"idx_nfe_manifestations_company_chave", "idx_company_nfe_documents_viewed", "idx_sync_requests_window"} {
		if n := countIndex(name); n != 1 {
			t.Errorf("index %s after down = %d, want 1", name, n)
		}
	}
}

func TestMigration014DropsTheCompaniesInitialSyncMirror(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, filepath.Join(t.TempDir(), "migrate.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrations, err := store.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 13); err != nil {
		t.Fatalf("migrate to version 13: %v", err)
	}

	const syncedAt = "2026-06-01T10:00:00Z"
	const mirrorOnlyAt = "2026-05-01T10:00:00Z"
	const now = "2026-09-01T10:00:00Z"
	insertCompany := `
		INSERT INTO companies (id, cnpj, cnpj_root, name, environment, sync_start_policy, initial_sync_completed_at, created_at, updated_at)
		VALUES (?, ?, '11222333', ?, 'producao', 'all', ?, ?, ?)
	`
	mustExec(t, db, insertCompany, "comp-1", "11222333000181", "A synced", syncedAt, now, now)
	mustExec(t, db, insertCompany, "comp-2", "11222333000262", "B mirror only", mirrorOnlyAt, now, now)
	mustExec(t, db, insertCompany, "comp-3", "11222333000343", "C never synced", nil, now, now)
	mustExec(t, db, `
		INSERT INTO company_sync_sources (company_id, source, initial_sync_completed_at, updated_at)
		VALUES ('comp-1', 'nfse', ?, ?), ('comp-3', 'nfe', ?, ?)
	`, syncedAt, now, syncedAt, now)

	if _, err := provider.UpTo(ctx, 14); err != nil {
		t.Fatalf("migrate to version 14: %v", err)
	}
	var mirrorColumns int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('companies') WHERE name = 'initial_sync_completed_at'`).Scan(&mirrorColumns); err != nil {
		t.Fatal(err)
	}
	if mirrorColumns != 0 {
		t.Error("companies.initial_sync_completed_at still exists after up")
	}

	// The repository query needs the latest schema.
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	companies, err := store.NewCompanyRepository(db).ListCompanies(ctx)
	if err != nil {
		t.Fatalf("list companies after up: %v", err)
	}
	wantDone := map[dfe.CompanyID]string{"comp-1": syncedAt, "comp-2": mirrorOnlyAt, "comp-3": ""}
	if len(companies) != len(wantDone) {
		t.Fatalf("companies after up = %d, want %d", len(companies), len(wantDone))
	}
	for _, c := range companies {
		var got string
		if c.InitialSyncDoneAt != nil {
			got = c.InitialSyncDoneAt.Format(time.RFC3339)
		}
		if got != wantDone[c.ID] {
			t.Errorf("%s NFS-e initial sync = %q, want %q", c.ID, got, wantDone[c.ID])
		}
	}

	if _, err := provider.DownTo(ctx, 13); err != nil {
		t.Fatalf("migrate down to version 13: %v", err)
	}
	for id, want := range wantDone {
		var got sql.NullString
		if err := db.QueryRowContext(ctx, `SELECT initial_sync_completed_at FROM companies WHERE id = ?`, string(id)).Scan(&got); err != nil {
			t.Fatalf("read companies.initial_sync_completed_at after down: %v", err)
		}
		if got.String != want {
			t.Errorf("%s companies.initial_sync_completed_at after down = %v, want %q", id, got, want)
		}
	}
}

func TestMigration016BackfillsNFeTpAmb(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, filepath.Join(t.TempDir(), "migrate.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrations, err := store.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 14); err != nil {
		t.Fatalf("migrate to version 14: %v", err)
	}

	const now = "2026-09-01T10:00:00Z"
	insertCompany := `
		INSERT INTO companies (id, cnpj, cnpj_root, name, environment, sync_start_policy, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'all', ?, ?)
	`
	mustExec(t, db, insertCompany, "comp-prod", "11222333000181", "11222333", "Produção", "producao", now, now)
	mustExec(t, db, insertCompany, "comp-hom", "44555666000199", "44555666", "Homologação", "producao_restrita", now, now)
	insertDocument := `
		INSERT INTO nfe_documents (id, chave_acesso, modelo, serie, numero, issue_date, competence, protocolo,
			emitente_cnpj, emitente_name, emitente_ie, emitente_uf, destinatario_cnpj, destinatario_name, transportador_cnpj,
			tp_nf, fin_nfe, nat_op, situacao, completeness, layout_version, raw_hash, created_at, updated_at)
		VALUES (?, ?, '55', '1', '1', ?, '2026-09', '', '', '', '', 'SP', '', '', '', '1', '', '', 'autorizada', 'resumo', '1.01', ?, ?, ?)
	`
	mustExec(t, db, insertDocument, "doc-prod", "chave-prod", now, "hash-prod", now, now)
	mustExec(t, db, insertDocument, "doc-hom", "chave-hom", now, "hash-hom", now, now)
	mustExec(t, db, insertDocument, "doc-orphan", "chave-orphan", now, "hash-orphan", now, now)
	insertRelation := `
		INSERT INTO company_nfe_documents (relation_id, company_id, nfe_document_id, company_role, visibility_reason, first_synced_at, last_synced_at)
		VALUES (?, ?, ?, 'destinatario', 'resumo_destinatario', ?, ?)
	`
	mustExec(t, db, insertRelation, "rel-prod", "comp-prod", "doc-prod", now, now)
	mustExec(t, db, insertRelation, "rel-hom", "comp-hom", "doc-hom", now, now)
	insertEvent := `
		INSERT INTO nfe_events (id, chave_acesso, tp_evento, type, n_seq_evento, protocolo, autor_cnpj, description,
			justificativa, correcao, completeness, raw_hash, created_at, updated_at)
		VALUES (?, ?, '210210', 'ciencia', 1, '', '', '', '', '', 'completa', ?, ?, ?)
	`
	mustExec(t, db, insertEvent, "ev-prod", "chave-prod", "ev-hash-prod", now, now)
	mustExec(t, db, insertEvent, "ev-hom", "chave-hom", "ev-hash-hom", now, now)
	mustExec(t, db, insertEvent, "ev-no-document", "chave-missing", "ev-hash-missing", now, now)

	if _, err := provider.UpTo(ctx, 16); err != nil {
		t.Fatalf("migrate to version 16: %v", err)
	}
	for table, want := range map[string]map[string]string{
		"nfe_documents": {"doc-prod": "1", "doc-hom": "2", "doc-orphan": ""},
		"nfe_events":    {"ev-prod": "1", "ev-hom": "2", "ev-no-document": ""},
	} {
		for id, wantTpAmb := range want {
			var got string
			if err := db.QueryRowContext(ctx, `SELECT tp_amb FROM `+table+` WHERE id = ?`, id).Scan(&got); err != nil { // #nosec G202 -- fixed table names.
				t.Fatalf("read %s.tp_amb: %v", table, err)
			}
			if got != wantTpAmb {
				t.Errorf("%s %s tp_amb = %q, want %q", table, id, got, wantTpAmb)
			}
		}
	}
	if _, err := db.ExecContext(ctx, `UPDATE nfe_documents SET tp_amb = '3' WHERE id = 'doc-prod'`); err == nil {
		t.Error("nfe_documents accepted tp_amb 3")
	}

	if _, err := provider.DownTo(ctx, 14); err != nil {
		t.Fatalf("migrate down to version 14: %v", err)
	}
	var columns int
	if err := db.QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM pragma_table_info('nfe_documents') WHERE name = 'tp_amb')
			+ (SELECT COUNT(*) FROM pragma_table_info('nfe_events') WHERE name = 'tp_amb')
	`).Scan(&columns); err != nil {
		t.Fatal(err)
	}
	if columns != 0 {
		t.Errorf("tp_amb columns after down = %d, want 0", columns)
	}
}

func TestMigration017AddsCTeViewedAt(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, filepath.Join(t.TempDir(), "migrate.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrations, err := store.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		t.Fatal(err)
	}
	viewedAt := func() (columns, indexes int) {
		t.Helper()
		if err := db.QueryRowContext(ctx, `
			SELECT
				(SELECT COUNT(*) FROM pragma_table_info('company_cte_documents') WHERE name = 'viewed_at'),
				(SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_company_cte_documents_viewed_at')
		`).Scan(&columns, &indexes); err != nil {
			t.Fatal(err)
		}
		return columns, indexes
	}

	if _, err := provider.UpTo(ctx, 16); err != nil {
		t.Fatalf("migrate to version 16: %v", err)
	}
	if columns, indexes := viewedAt(); columns != 0 || indexes != 0 {
		t.Fatalf("before 017: viewed_at columns = %d, indexes = %d, want 0 and 0", columns, indexes)
	}

	if _, err := provider.UpTo(ctx, 17); err != nil {
		t.Fatalf("migrate to version 17: %v", err)
	}
	if columns, indexes := viewedAt(); columns != 1 || indexes != 1 {
		t.Fatalf("after 017: viewed_at columns = %d, indexes = %d, want 1 and 1", columns, indexes)
	}
	if _, err := provider.DownTo(ctx, 16); err != nil {
		t.Fatalf("migrate down to version 16: %v", err)
	}
	if columns, indexes := viewedAt(); columns != 0 || indexes != 0 {
		t.Errorf("after down: viewed_at columns = %d, indexes = %d, want 0 and 0", columns, indexes)
	}
}

func TestMigration018CountsExportedEvents(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, filepath.Join(t.TempDir(), "migrate.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrations, err := store.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 17); err != nil {
		t.Fatalf("migrate to version 17: %v", err)
	}

	const before = "2026-09-01T09:00:00Z"
	const exportedAt = "2026-09-01T10:00:00Z"
	const nfeChave = "35260911222333000181550010000012341123456787"
	const cteChave = "35260912345678000195570010000001011123456784"
	mustExec(t, db, `
		INSERT INTO companies (id, cnpj, cnpj_root, name, environment, sync_start_policy, created_at, updated_at)
		VALUES ('comp-1', '70860312000150', '70860312', 'Company', 'producao', 'all', ?, ?)
	`, before, before)

	mustExec(t, db, `
		INSERT INTO nfe_documents (id, chave_acesso, modelo, serie, numero, issue_date, competence, protocolo,
			emitente_cnpj, emitente_name, emitente_ie, emitente_uf, destinatario_cnpj, destinatario_name, transportador_cnpj,
			tp_nf, fin_nfe, nat_op, situacao, completeness, layout_version, raw_hash, created_at, updated_at)
		VALUES ('nfe-1', ?, '55', '1', '1', ?, '2026-09', '', '', '', '', 'SP', '', '', '', '1', '', '', 'autorizada', 'completa', '4.00', 'hash-nfe', ?, ?)
	`, nfeChave, before, before, before)
	insertNFeEvent := `
		INSERT INTO nfe_events (id, chave_acesso, tp_evento, type, n_seq_evento, protocolo, autor_cnpj, description,
			justificativa, correcao, completeness, raw_hash, created_at, updated_at)
		VALUES (?, ?, ?, 'ciencia', 1, '', '', '', '', '', 'completa', ?, ?, ?)
	`
	mustExec(t, db, insertNFeEvent, "nfe-ev-before", nfeChave, "210210", "hash-ev-1", before, before)
	mustExec(t, db, insertNFeEvent, "nfe-ev-same-second", nfeChave, "210200", "hash-ev-2", exportedAt, exportedAt)
	mustExec(t, db, `
		INSERT INTO company_nfe_export_marks (company_id, nfe_document_id, export_kind, exported_hash, exported_at)
		VALUES ('comp-1', 'nfe-1', 'xml', 'hash-nfe', ?)
	`, exportedAt)

	mustExec(t, db, `
		INSERT INTO cte_documents (id, chave_acesso, tp_amb, modelo, tipo_documento, serie, numero, cfop, nat_op,
			issue_date, competence, protocolo, tp_cte, tp_serv, modal,
			mun_ini_codigo, mun_ini_nome, uf_ini, mun_fim_codigo, mun_fim_nome, uf_fim,
			emitente_cnpj, emitente_name, emitente_ie, emitente_uf, remetente_cnpj, remetente_name,
			destinatario_cnpj, destinatario_name, expedidor_cnpj, expedidor_name, recebedor_cnpj, recebedor_name,
			tomador_indicador, tomador_cnpj, tomador_name, tomador_ie, tomador_uf,
			produto_predominante, situacao, layout_version, raw_hash, created_at, updated_at)
		VALUES ('cte-1', ?, '1', '57', 'cte', '1', '101', '6353', '', ?, '2026-09', '', '0', '0', '01',
			'', '', 'SP', '', '', 'RJ', '12345678000195', 'Transportadora', '', 'SP', '', '', '', '', '', '', '', '',
			'3', '70860312000150', 'Company', '', 'RJ', '', 'autorizada', '4.00', 'hash-cte', ?, ?)
	`, cteChave, before, before, before)
	insertCTeEvent := `
		INSERT INTO cte_events (id, chave_acesso, tp_amb, c_orgao, tp_evento, type, n_seq_evento, protocolo, autor_cnpj,
			description, justificativa, observacao, correcao, condicao_uso, raw_hash, created_at, updated_at)
		VALUES (?, ?, '1', '35', ?, 'carta_correcao', 1, '', '', '', '', '', '', '', ?, ?, ?)
	`
	mustExec(t, db, insertCTeEvent, "cte-ev-before", cteChave, "110110", "hash-cte-ev-1", before, before)
	mustExec(t, db, insertCTeEvent, "cte-ev-same-second", cteChave, "110180", "hash-cte-ev-2", exportedAt, exportedAt)
	mustExec(t, db, `
		INSERT INTO company_cte_export_marks (company_id, cte_document_id, export_kind, exported_hash, exported_at)
		VALUES ('comp-1', 'cte-1', 'xml', 'hash-cte', ?)
	`, exportedAt)

	if _, err := provider.UpTo(ctx, 18); err != nil {
		t.Fatalf("migrate to version 18: %v", err)
	}
	// Only the event stored strictly before the mark counts as exported, so
	// the one of the same second leaves the document pending once.
	for _, table := range []string{"company_nfe_export_marks", "company_cte_export_marks"} {
		var exported int
		if err := db.QueryRowContext(ctx, `SELECT exported_events FROM `+table).Scan(&exported); err != nil { // #nosec G202 -- fixed table names.
			t.Fatalf("read %s.exported_events: %v", table, err)
		}
		if exported != 1 {
			t.Errorf("%s exported_events = %d, want 1", table, exported)
		}
	}

	if _, err := provider.DownTo(ctx, 17); err != nil {
		t.Fatalf("migrate down to version 17: %v", err)
	}
	var columns int
	if err := db.QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM pragma_table_info('company_nfe_export_marks') WHERE name = 'exported_events')
			+ (SELECT COUNT(*) FROM pragma_table_info('company_cte_export_marks') WHERE name = 'exported_events')
	`).Scan(&columns); err != nil {
		t.Fatal(err)
	}
	if columns != 0 {
		t.Errorf("exported_events columns after down = %d, want 0", columns)
	}
}

func TestMigration019KeysSyncSourcesByEnvironment(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, filepath.Join(t.TempDir(), "migrate.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrations, err := store.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 18); err != nil {
		t.Fatalf("migrate to version 18: %v", err)
	}

	const now = "2026-09-01T10:00:00Z"
	const blockedUntil = "2026-09-01T11:00:00Z"
	insertCompany := `
		INSERT INTO companies (id, cnpj, cnpj_root, name, environment, sync_start_policy, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'all', ?, ?)
	`
	mustExec(t, db, insertCompany, "comp-prod", "11222333000181", "11222333", "Produção", "producao", now, now)
	mustExec(t, db, insertCompany, "comp-hom", "44555666000199", "44555666", "Homologação", "producao_restrita", now, now)
	for _, companyID := range []string{"comp-prod", "comp-hom"} {
		mustExec(t, db, `
			INSERT INTO company_sync_sources (company_id, source, initial_sync_completed_at, blocked_until, blocked_reason, updated_at)
			VALUES (?, 'nfe', ?, ?, 'consumo_indevido', ?)
		`, companyID, now, blockedUntil, now)
		mustExec(t, db, `INSERT INTO sync_requests (company_id, source, requested_at) VALUES (?, 'nfe', ?)`, companyID, now)
	}

	// environments reads the environment of each company's rows in table.
	environments := func(table string) map[string]string {
		t.Helper()
		rows, err := db.QueryContext(ctx, `SELECT company_id, environment FROM `+table) // #nosec G202 -- fixed table names.
		if err != nil {
			t.Fatalf("read %s: %v", table, err)
		}
		defer func() { _ = rows.Close() }()
		got := map[string]string{}
		for rows.Next() {
			var companyID, environment string
			if err := rows.Scan(&companyID, &environment); err != nil {
				t.Fatal(err)
			}
			got[companyID] += environment
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		return got
	}
	countRows := func(query string) int {
		t.Helper()
		var n int
		if err := db.QueryRowContext(ctx, query).Scan(&n); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		return n
	}

	if _, err := provider.UpTo(ctx, 19); err != nil {
		t.Fatalf("migrate to version 19: %v", err)
	}
	want := map[string]string{"comp-prod": "producao", "comp-hom": "producao_restrita"}
	for _, table := range []string{"company_sync_sources", "sync_requests"} {
		got := environments(table)
		if len(got) != len(want) || got["comp-prod"] != want["comp-prod"] || got["comp-hom"] != want["comp-hom"] {
			t.Errorf("%s environments after up = %v, want %v", table, got, want)
		}
	}
	var until, reason string
	if err := db.QueryRowContext(ctx, `SELECT blocked_until, blocked_reason FROM company_sync_sources WHERE company_id = 'comp-prod'`).Scan(&until, &reason); err != nil {
		t.Fatal(err)
	}
	if until != blockedUntil || reason != "consumo_indevido" {
		t.Errorf("comp-prod block after up = (%s, %s), want (%s, consumo_indevido)", until, reason, blockedUntil)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO sync_requests (company_id, source, environment, requested_at) VALUES ('comp-prod', 'nfe', 'bogus', ?)`, now); err == nil {
		t.Error("sync_requests accepted environment 'bogus', want a CHECK failure")
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO company_sync_sources (company_id, source, environment, updated_at) VALUES ('comp-prod', 'cte', 'bogus', ?)`, now); err == nil {
		t.Error("company_sync_sources accepted environment 'bogus', want a CHECK failure")
	}
	if n := countRows(`SELECT COUNT(*) FROM pragma_index_info('idx_sync_requests_window') WHERE name = 'environment'`); n != 1 {
		t.Errorf("idx_sync_requests_window environment columns after up = %d, want 1", n)
	}

	// Rows of the environment the company is not in now are dropped by Down.
	mustExec(t, db, `
		INSERT INTO company_sync_sources (company_id, source, environment, blocked_until, blocked_reason, updated_at)
		VALUES ('comp-prod', 'nfe', 'producao_restrita', ?, 'rate_budget', ?)
	`, blockedUntil, now)
	mustExec(t, db, `INSERT INTO sync_requests (company_id, source, environment, requested_at) VALUES ('comp-prod', 'nfe', 'producao_restrita', ?)`, now)

	if _, err := provider.DownTo(ctx, 18); err != nil {
		t.Fatalf("migrate down to version 18: %v", err)
	}
	for _, table := range []string{"company_sync_sources", "sync_requests"} {
		if n := countRows(`SELECT COUNT(*) FROM pragma_table_info('` + table + `') WHERE name = 'environment'`); n != 0 {
			t.Errorf("%s environment columns after down = %d, want 0", table, n)
		}
		if n := countRows(`SELECT COUNT(*) FROM ` + table); n != 2 { // #nosec G202 -- fixed table names.
			t.Errorf("%s rows after down = %d, want 2", table, n)
		}
	}
	if err := db.QueryRowContext(ctx, `SELECT blocked_reason FROM company_sync_sources WHERE company_id = 'comp-prod'`).Scan(&reason); err != nil {
		t.Fatal(err)
	}
	if reason != "consumo_indevido" {
		t.Errorf("comp-prod block reason after down = %s, want the produção one", reason)
	}
	if n := countRows(`SELECT COUNT(*) FROM pragma_index_info('idx_sync_requests_window')`); n != 3 {
		t.Errorf("idx_sync_requests_window columns after down = %d, want 3", n)
	}
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func TestMigration015AddsCTeTables(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, filepath.Join(t.TempDir(), "migrate.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrations, err := store.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 14); err != nil {
		t.Fatalf("migrate to version 14: %v", err)
	}
	const now = "2026-09-01T10:00:00Z"
	mustExec(t, db, `
		INSERT INTO companies (id, cnpj, cnpj_root, name, environment, sync_start_policy, created_at, updated_at)
		VALUES ('comp-1', '70860312000150', '70860312', 'Company', 'producao', 'all', ?, ?)
	`, now, now)

	cteTables := []string{"cte_documents", "company_cte_documents", "cte_events", "company_cte_export_marks"}
	countTables := func() int {
		t.Helper()
		var n int
		err := db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM sqlite_master
			WHERE type = 'table' AND name IN (?, ?, ?, ?)
		`, cteTables[0], cteTables[1], cteTables[2], cteTables[3]).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}

	if _, err := provider.UpTo(ctx, 15); err != nil {
		t.Fatalf("migrate to version 15: %v", err)
	}
	if n := countTables(); n != len(cteTables) {
		t.Errorf("CT-e tables after up = %d, want %d", n, len(cteTables))
	}

	insertDocument := `
		INSERT INTO cte_documents (id, chave_acesso, tp_amb, modelo, tipo_documento, serie, numero, cfop, nat_op,
			issue_date, competence, protocolo, tp_cte, tp_serv, modal,
			mun_ini_codigo, mun_ini_nome, uf_ini, mun_fim_codigo, mun_fim_nome, uf_fim,
			emitente_cnpj, emitente_name, emitente_ie, emitente_uf, remetente_cnpj, remetente_name,
			destinatario_cnpj, destinatario_name, expedidor_cnpj, expedidor_name, recebedor_cnpj, recebedor_name,
			tomador_indicador, tomador_cnpj, tomador_name, tomador_ie, tomador_uf,
			produto_predominante, situacao, layout_version, raw_hash, created_at, updated_at)
		VALUES (?, ?, '1', '57', 'cte', '1', '101', '6353', '', ?, '2026-09', '', '0', '0', '01',
			'', '', 'SP', '', '', 'RJ', '12345678000195', 'Transportadora', '', 'SP', '', '', '', '', '', '', '', '',
			'3', '70860312000150', 'Company', '', 'RJ', '', ?, '4.00', ?, ?, ?)
	`
	mustExec(t, db, insertDocument, "doc-1", "35260912345678000195570010000001011123456784", now, "autorizada", "hash-1", now, now)
	if _, err := db.ExecContext(ctx, insertDocument, "doc-2", "35260912345678000195570010000001021234567891", now, "autorizado", "hash-2", now, now); err == nil {
		t.Error("cte_documents accepted situacao 'autorizado', want a CHECK failure")
	}
	mustExec(t, db, `
		INSERT INTO company_cte_documents (relation_id, company_id, cte_document_id, company_role, papeis,
			visibility_reason, first_synced_at, last_synced_at)
		VALUES ('rel-1', 'comp-1', 'doc-1', 'tomador', 'tomador,destinatario', 'exact_tomador', ?, ?)
	`, now, now)
	var autorizados, nfeChaves string
	var maskedKeys int
	err = db.QueryRowContext(ctx, `SELECT autorizados_cnpj, nfe_chaves, masked_keys FROM cte_documents WHERE id = 'doc-1'`).
		Scan(&autorizados, &nfeChaves, &maskedKeys)
	if err != nil {
		t.Fatal(err)
	}
	if autorizados != "" || nfeChaves != "" || maskedKeys != 0 {
		t.Errorf("default (autorizados_cnpj, nfe_chaves, masked_keys) = (%q, %q, %d), want empty", autorizados, nfeChaves, maskedKeys)
	}

	if _, err := provider.DownTo(ctx, 14); err != nil {
		t.Fatalf("migrate down to version 14: %v", err)
	}
	if n := countTables(); n != 0 {
		t.Errorf("CT-e tables after down = %d, want 0", n)
	}
	var companies int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM companies`).Scan(&companies); err != nil {
		t.Fatal(err)
	}
	if companies != 1 {
		t.Errorf("companies after down = %d, want 1", companies)
	}
}

func TestMigration020StripsTheNFSPrefixOfNFSeChaves(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDB(ctx, filepath.Join(t.TempDir(), "migrate.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrations, err := store.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 19); err != nil {
		t.Fatalf("migrate to version 19: %v", err)
	}

	const now = "2026-09-01T10:00:00Z"
	const chaveX = "35503082245852546000109000000000000126060000000011"
	const chaveY = "35503082245852546000109000000000000226060000000022"
	const invalidID = "NFS3550308224585254600010900000000000032606000000003X"
	insertDoc := `
		INSERT INTO documents (
			id, chave_acesso, issue_date, competence, created_at, updated_at,
			prestador_cnpj, prestador_name, tomador_cnpj, tomador_name, intermediario_cnpj, intermediario_name,
			status, layout_version, xml_path, raw_hash, nfse_number, service_description
		) VALUES (?, ?, ?, '2026-09', ?, ?, '45852546000109', 'P', '11222333000181', 'T', '', '',
			'normal', '1.0', '', ?, '1', 'Serviço')`
	// (a) prefixed, no collision, with a cancelamento never linked to it
	mustExec(t, db, insertDoc, "doc-a", "NFS"+chaveX, now, now, now, "hash-a")
	mustExec(t, db, `
		INSERT INTO events (id, document_id, chave_acesso, type, event_at, replacement_chave_acesso,
			description, raw_xml_path, raw_hash, created_at)
		VALUES ('ev-a', NULL, ?, 'cancelamento', ?, '', 'Cancelamento', '', 'hash-ev-a', ?)
	`, chaveX, now, now)
	// (b) prefixed, but another document already has the 50 digits
	mustExec(t, db, insertDoc, "doc-b", "NFS"+chaveY, now, now, now, "hash-b")
	mustExec(t, db, insertDoc, "doc-y", chaveY, now, now, now, "hash-y")
	// (c) an Id that is not "NFS" + 50 digits
	mustExec(t, db, insertDoc, "doc-c", invalidID, now, now, now, "hash-c")

	if _, err := provider.UpTo(ctx, 20); err != nil {
		t.Fatalf("migrate to version 20: %v", err)
	}

	docChaveAndStatus := func(id string) (string, string) {
		t.Helper()
		var chave, status string
		if err := db.QueryRowContext(ctx, `SELECT chave_acesso, status FROM documents WHERE id = ?`, id).Scan(&chave, &status); err != nil {
			t.Fatalf("read %s: %v", id, err)
		}
		return chave, status
	}
	if chave, status := docChaveAndStatus("doc-a"); chave != chaveX || status != "cancelada" {
		t.Errorf("doc-a = (%q, %q), want (%q, cancelada)", chave, status, chaveX)
	}
	var linked sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT document_id FROM events WHERE id = 'ev-a'`).Scan(&linked); err != nil {
		t.Fatalf("read ev-a: %v", err)
	}
	if linked.String != "doc-a" {
		t.Errorf("ev-a document_id = %q, want doc-a", linked.String)
	}
	if chave, status := docChaveAndStatus("doc-b"); chave != "NFS"+chaveY || status != "normal" {
		t.Errorf("doc-b = (%q, %q), want the prefixed chave untouched", chave, status)
	}
	if chave, _ := docChaveAndStatus("doc-y"); chave != chaveY {
		t.Errorf("doc-y chave = %q, want %q", chave, chaveY)
	}
	if chave, _ := docChaveAndStatus("doc-c"); chave != invalidID {
		t.Errorf("doc-c chave = %q, want %q", chave, invalidID)
	}

	if _, err := provider.DownTo(ctx, 19); err != nil {
		t.Fatalf("migrate down to version 19: %v", err)
	}
	if chave, _ := docChaveAndStatus("doc-a"); chave != chaveX {
		t.Errorf("doc-a chave after down = %q, want the 50 digits kept", chave)
	}
}

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
	if err := db.QueryRowContext(ctx, `SELECT autorizados_cnpj, nfe_chaves FROM cte_documents WHERE id = 'doc-1'`).Scan(&autorizados, &nfeChaves); err != nil {
		t.Fatal(err)
	}
	if autorizados != "" || nfeChaves != "" {
		t.Errorf("default (autorizados_cnpj, nfe_chaves) = (%q, %q), want empty", autorizados, nfeChaves)
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

package store_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"

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

func TestMigration011AddsManifestationTpAmb(t *testing.T) {
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

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

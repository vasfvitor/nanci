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

	if _, err := provider.Down(ctx); err != nil {
		t.Fatalf("migrate down: %v", err)
	}
	var stateRows int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_state WHERE company_id = 'comp-1' AND last_checked_nsu = 42`).Scan(&stateRows); err != nil {
		t.Fatalf("read sync_state after down: %v", err)
	}
	if stateRows != 1 {
		t.Errorf("sync_state rows after down = %d, want 1", stateRows)
	}
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

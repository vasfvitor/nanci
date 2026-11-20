package cli

import (
	"context"
	"strings"
	"testing"
)

// seedNFSeCursors gives the test company an NFS-e cursor at NSU 7 and an
// NF-e cursor, so a test can check that the NFS-e reset leaves the NF-e one.
func seedNFSeCursors(t *testing.T, env *nfeTestRoot) {
	t.Helper()
	const now = "2026-11-20T10:00:00Z"
	for _, source := range []string{"nfse", "nfe"} {
		if _, err := env.db.ExecContext(context.Background(), `
			INSERT INTO sync_state (company_id, source, environment, consultation_cnpj, last_checked_nsu, created_at, updated_at)
			VALUES (?, ?, 'producao', ?, 7, ?, ?)
		`, string(env.company.ID), source, env.company.CNPJ, now, now); err != nil {
			t.Fatal(err)
		}
	}
}

func syncCursorCount(t *testing.T, env *nfeTestRoot, source string) int {
	t.Helper()
	var n int
	if err := env.db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM sync_state WHERE company_id = ? AND source = ?`,
		string(env.company.ID), source,
	).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestReset_DryRunThenConfirm(t *testing.T) {
	env := newNFeTestRoot(t)
	seedNFSeCursors(t, env)

	if err := env.run("reset", "-c", nfeTestCNPJ); err != nil {
		t.Fatalf("reset dry-run: %v", err)
	}
	got := env.out.String()
	for _, want := range []string{"Redefinição da NFS-e para: Empresa Mock", "Último NSU consultado: 7", "Nada foi alterado. Use --confirmar"} {
		if !strings.Contains(got, want) {
			t.Errorf("dry-run output lacks %q:\n%s", want, got)
		}
	}
	if n := syncCursorCount(t, env, "nfse"); n != 1 {
		t.Errorf("NFS-e cursors after the dry run = %d, want 1", n)
	}
	if env.passwords.requests != 0 {
		t.Errorf("password requests = %d, want 0", env.passwords.requests)
	}

	if err := env.run("reset", "-c", nfeTestCNPJ, "--confirmar"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if got := env.out.String(); !strings.Contains(got, "NFS-e redefinida para: Empresa Mock") || !strings.Contains(got, "nanci company update") {
		t.Errorf("reset output:\n%s", got)
	}
	if n := syncCursorCount(t, env, "nfse"); n != 0 {
		t.Errorf("NFS-e cursors after the reset = %d, want 0", n)
	}
	if n := syncCursorCount(t, env, "nfe"); n != 1 {
		t.Errorf("the NFS-e reset removed the NF-e cursor: %d left, want 1", n)
	}
	if env.passwords.requests != 0 {
		t.Errorf("password requests = %d, want 0", env.passwords.requests)
	}
}

func TestReset_JSON(t *testing.T) {
	env := newNFeTestRoot(t)
	seedNFSeCursors(t, env)

	if err := env.run("reset", "-c", nfeTestCNPJ, "--json"); err != nil {
		t.Fatalf("reset --json: %v", err)
	}
	var preview map[string]any
	decodeJSONOutput(t, env.out, &preview)
	if preview["Confirmed"] != false || preview["LastProcessedNSU"] != float64(7) || preview["Environment"] != "producao" || preview["CNPJ"] != nfeTestCNPJ {
		t.Errorf("preview = %v", preview)
	}
	assertNoJSONKeys(t, preview, internalJSONKeys...)

	if err := env.run("reset", "-c", nfeTestCNPJ, "--confirmar", "--json"); err != nil {
		t.Fatalf("reset --confirmar --json: %v", err)
	}
	var result map[string]any
	decodeJSONOutput(t, env.out, &result)
	if result["Confirmed"] != true || result["LastProcessedNSU"] != float64(7) {
		t.Errorf("result = %v", result)
	}
	if n := syncCursorCount(t, env, "nfse"); n != 0 {
		t.Errorf("NFS-e cursors after the reset = %d, want 0", n)
	}
}

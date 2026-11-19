package cli

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/sync"
)

// seedNFSe stores an NFS-e the company provided, issued on 2026-06-10.
func (e *nfeTestRoot) seedNFSe(id, chave string) {
	e.t.Helper()
	ctx := context.Background()
	const now = "2026-06-10T00:00:00Z"
	if _, err := e.db.ExecContext(ctx, `
		INSERT INTO documents (
			id, chave_acesso, issue_date, competence, created_at, updated_at,
			prestador_cnpj, prestador_name, tomador_cnpj, tomador_name, intermediario_cnpj, intermediario_name,
			status, layout_version, xml_path, raw_hash, nfse_number, service_description
		)
		VALUES (?, ?, ?, '2026-06', ?, ?, ?, 'P', '11222333000181', 'T', '', '', 'normal', '1.0', '', ?, '1', 'Serviço')
	`, id, chave, now, now, now, nfeTestCNPJ, "hash-"+id); err != nil {
		e.t.Fatal(err)
	}
	if _, err := e.db.ExecContext(ctx, `
		INSERT INTO company_documents (relation_id, company_id, document_id, company_role, visibility_reason, first_synced_at, last_synced_at)
		VALUES (?, ?, ?, 'prestada', 'exact_prestador', ?, ?)
	`, "rel-"+id, string(e.company.ID), id, now, now); err != nil {
		e.t.Fatal(err)
	}
}

func TestList_NaoVistos(t *testing.T) {
	const (
		chaveVista = "35503082270860312000150000000000000126060000000011"
		chaveNova  = "35503082270860312000150000000000000226060000000022"
	)
	env := newNFeTestRoot(t)
	// List every NFS-e, not only those issued after the company was added.
	if _, err := env.db.ExecContext(context.Background(), `UPDATE companies SET sync_start_policy = 'all', sync_start_date = NULL WHERE id = ?`, string(env.company.ID)); err != nil {
		t.Fatal(err)
	}
	env.seedNFSe("doc-vista", chaveVista)
	env.seedNFSe("doc-nova", chaveNova)
	marked, err := store.NewDocumentRepository(env.db).MarkViewed(context.Background(), env.company.ID, []string{chaveVista})
	if err != nil || marked != 1 {
		t.Fatalf("MarkViewed = %d, %v; want 1", marked, err)
	}

	if err := env.run("list", "-c", nfeTestCNPJ); err != nil {
		t.Fatalf("list: %v", err)
	}
	if got := env.out.String(); !strings.Contains(got, chaveVista) || !strings.Contains(got, chaveNova) || !strings.Contains(got, "Total de 2 documento(s) listado(s).") {
		t.Errorf("list without --nao-vistos:\n%s", got)
	}

	if err := env.run("list", "-c", nfeTestCNPJ, "--nao-vistos"); err != nil {
		t.Fatalf("list --nao-vistos: %v", err)
	}
	if got := env.out.String(); strings.Contains(got, chaveVista) || !strings.Contains(got, chaveNova) || !strings.Contains(got, "Total de 1 documento(s) listado(s).") {
		t.Errorf("list --nao-vistos:\n%s", got)
	}
}

func TestList_JSON(t *testing.T) {
	const chave = "35503082270860312000150000000000000126060000000011"
	env := newNFeTestRoot(t)
	if _, err := env.db.ExecContext(context.Background(), `UPDATE companies SET sync_start_policy = 'all', sync_start_date = NULL WHERE id = ?`, string(env.company.ID)); err != nil {
		t.Fatal(err)
	}
	env.seedNFSe("doc-1", chave)

	if err := env.run("list", "-c", nfeTestCNPJ, "--json"); err != nil {
		t.Fatalf("list --json: %v", err)
	}
	var rows []map[string]any
	decodeJSONOutput(t, env.out, &rows)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1:\n%s", len(rows), env.out.String())
	}
	row := rows[0]
	if row["ChaveAcesso"] != chave || row["CompanyRole"] != "prestada" || row["PrestadorCNPJ"] != nfeTestCNPJ || row["ViewedAt"] != nil {
		t.Errorf("row = %v", row)
	}
	if _, ok := row["ServiceValue"].(float64); !ok {
		t.Errorf("ServiceValue = %v, want a number", row["ServiceValue"])
	}
	if warnings, ok := row["ParseWarnings"].([]any); !ok || len(warnings) != 0 {
		t.Errorf("ParseWarnings = %v, want []", row["ParseWarnings"])
	}
	assertNoJSONKeys(t, rows, internalJSONKeys...)
	assertNoJSONKeys(t, rows, "ID")

	env = newNFeTestRoot(t) // flag values stick to a command tree
	if err := env.run("list", "-c", nfeTestCNPJ, "--json"); err != nil {
		t.Fatalf("empty list --json: %v", err)
	}
	if got := env.out.String(); got != "[]\n" {
		t.Errorf("empty list = %q, want []", got)
	}
}

func TestStatus_JSON(t *testing.T) {
	env := newNFeTestRoot(t)
	if err := env.run("status", "-c", nfeTestCNPJ, "--json"); err != nil {
		t.Fatalf("status --json: %v", err)
	}
	var status map[string]any
	decodeJSONOutput(t, env.out, &status)
	if status["CNPJ"] != nfeTestCNPJ || status["TotalEmitidas"] != float64(0) || status["LastFoundNSU"] != nil {
		t.Errorf("status = %v", status)
	}
}

func TestPull_JSONFailurePrintsNothing(t *testing.T) {
	// The refusing provider fails the NFS-e pull at the certificate.
	env := newNFeTestRoot(t)
	if err := env.run("pull", "-c", nfeTestCNPJ, "--json"); err == nil {
		t.Fatal("pull without a certificate password succeeded")
	}
	if got := env.out.String(); got != "" {
		t.Errorf("stdout = %q, want nothing", got)
	}
}

func TestNFSePullOutput(t *testing.T) {
	found := int64(7)
	got := newNFSePullOutput(sync.PullResult{
		CompanyName:      "Empresa",
		CNPJ:             nfeTestCNPJ,
		CredentialLabel:  "Mock A1",
		Status:           "completed",
		LastProcessedNSU: 9,
		LastFoundNSU:     &found,
		DocumentsFound:   3,
		Duration:         1500 * time.Millisecond,
	})
	if got.DurationMs != 1500 || got.LastProcessedNSU != 9 || *got.LastFoundNSU != 7 || got.DocumentsFound != 3 || got.Status != "completed" {
		t.Errorf("newNFSePullOutput = %+v", got)
	}
}

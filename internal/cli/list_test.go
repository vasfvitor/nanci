package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
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
	marked, err := store.NewDocumentRepository(env.db).MarkDocumentsViewed(context.Background(), env.company.ID, nfse.DocumentFilter{ChavesAcesso: []string{chaveVista}})
	if err != nil || marked != 1 {
		t.Fatalf("MarkDocumentsViewed = %d, %v; want 1", marked, err)
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

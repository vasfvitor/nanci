package store_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/store/storetest"
)

const (
	nfseChaveA = "35503082245852546000109000000000000126060000000011"
	nfseChaveB = "35503082245852546000109000000000000226060000000022"
	nfseChaveC = "35503082245852546000109000000000000326060000000033"
)

// newNFSeFixture stores three NFS-e seen by two companies, comp-a and comp-b.
func newNFSeFixture(t *testing.T) *store.DocumentRepository {
	t.Helper()
	db := storetest.OpenTestDB(t)
	ctx := context.Background()
	cred := storetest.TestCredential("cred-1")
	if err := credential.NewStore(db).CreateCredential(ctx, cred); err != nil {
		t.Fatal(err)
	}
	companies := company.NewStore(db)
	for id, cnpj := range map[string]string{"comp-a": "45852546000109", "comp-b": "11222333000181"} {
		if err := companies.CreateCompany(ctx, storetest.TestCompany(id, cnpj, dfe.EnvironmentProduction, cred)); err != nil {
			t.Fatal(err)
		}
	}
	for i, chave := range []string{nfseChaveA, nfseChaveB, nfseChaveC} {
		id := "doc-" + chave[len(chave)-2:]
		if _, err := db.ExecContext(ctx, `
			INSERT INTO documents (
				id, chave_acesso, issue_date, competence, created_at, updated_at,
				prestador_cnpj, prestador_name, tomador_cnpj, tomador_name, intermediario_cnpj, intermediario_name,
				status, layout_version, xml_path, raw_hash, nfse_number, service_description
			) VALUES (?, ?, ?, '2026-06', '2026-06-01T00:00:00Z', '2026-06-01T00:00:00Z',
				'45852546000109', 'P', '11222333000181', 'T', '', '',
				'normal', '1.0', '', ?, '1', 'Serviço')`,
			id, chave, fmt.Sprintf("2026-06-0%dT00:00:00Z", i+1), "hash-"+id); err != nil {
			t.Fatal(err)
		}
		for _, company := range []string{"comp-a", "comp-b"} {
			if _, err := db.ExecContext(ctx, `
				INSERT INTO company_documents (
					relation_id, company_id, document_id, company_role, visibility_reason, first_synced_at, last_synced_at
				) VALUES (?, ?, ?, 'prestada', 'exact_prestador', '2026-06-01T00:00:00Z', '2026-06-01T00:00:00Z')`,
				company+"-"+id, company, id); err != nil {
				t.Fatal(err)
			}
		}
	}
	return store.NewDocumentRepository(db)
}

func listNFSeChaves(t *testing.T, repo *store.DocumentRepository, companyID dfe.CompanyID, filter nfse.DocumentFilter) []string {
	t.Helper()
	docs, err := repo.ListCompanyDocuments(context.Background(), companyID, filter)
	if err != nil {
		t.Fatal(err)
	}
	var chaves []string
	for _, d := range docs {
		chaves = append(chaves, string(d.ChaveAcesso))
	}
	return chaves
}

func TestNFSeMarkViewed(t *testing.T) {
	ctx := context.Background()
	repo := newNFSeFixture(t)
	unread := nfse.DocumentFilter{OnlyUnread: true}

	// Marks the given chaves only, the NFS fallback chave included.
	n, err := repo.MarkViewed(ctx, "comp-a", []string{nfseChaveA, nfseChaveC})
	if err != nil || n != 2 {
		t.Fatalf("MarkViewed = %d, %v; want 2", n, err)
	}
	if got := listNFSeChaves(t, repo, "comp-a", unread); !slices.Equal(got, []string{nfseChaveB}) {
		t.Errorf("comp-a unread = %v, want only B", got)
	}

	// The mark belongs to the company.
	if got := listNFSeChaves(t, repo, "comp-b", unread); len(got) != 3 {
		t.Errorf("comp-b unread = %v, want all three", got)
	}

	// Already-viewed rows are not counted again; no chave marks nothing.
	if n, err := repo.MarkViewed(ctx, "comp-a", []string{nfseChaveA, nfseChaveB}); err != nil || n != 1 {
		t.Errorf("MarkViewed again = %d, %v; want 1", n, err)
	}
	if n, err := repo.MarkViewed(ctx, "comp-b", nil); err != nil || n != 0 {
		t.Errorf("MarkViewed without chaves = %d, %v; want 0", n, err)
	}
}

func TestNFSeFiltersByChaves(t *testing.T) {
	ctx := context.Background()
	repo := newNFSeFixture(t)
	filter := nfse.DocumentFilter{ChavesAcesso: []string{nfseChaveC, nfseChaveA}}

	if got := listNFSeChaves(t, repo, "comp-a", filter); !slices.Equal(got, []string{nfseChaveC, nfseChaveA}) {
		t.Errorf("listed by chaves = %v, want C and A, newest first", got)
	}

	pending, err := repo.ListPendingExportDocuments(ctx, "comp-a", filter, "xml")
	if err != nil {
		t.Fatal(err)
	}
	count, err := repo.CountPendingExportDocuments(ctx, "comp-a", filter, "xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 || count != 2 {
		t.Errorf("pending export by chaves = %d listed, %d counted; want 2 and 2", len(pending), count)
	}

	if err := repo.MarkDocumentsExported(ctx, "comp-a", "xml", []nfse.DocumentExportMark{{DocumentID: string(pending[0].DocumentID), Hash: pending[0].RawHash}}); err != nil {
		t.Fatal(err)
	}
	if count, err := repo.CountPendingExportDocuments(ctx, "comp-a", filter, "xml"); err != nil || count != 1 {
		t.Errorf("pending after one export = %d, %v; want 1", count, err)
	}
}

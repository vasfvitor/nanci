package app_test

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestAppIntegration_ListDocuments(t *testing.T) {
	application, db := setupTestApp(t)
	ctx := context.Background()

	certPath, _ := filepath.Abs("app_list_integration_test.go")
	if err := application.Credentials.AddCredential(ctx, credential.AddCredentialInput{Label: "L", CertPath: certPath}); err != nil {
		t.Fatalf("AddCredential: %v", err)
	}
	creds, _ := application.Credentials.ListCredentials(ctx)

	now := time.Now().Truncate(24 * time.Hour)
	policyFromNow, dateFromNow, _ := company.ParseSyncStartPolicyInput("from_now", "")

	// Create Company with from_now policy
	if err := application.Companies.AddCompany(ctx, company.AddCompanyInput{
		CNPJ:            "45852546000109",
		Name:            "Empresa Listagem",
		Environment:     nfse.EnvironmentRestricted,
		CredentialID:    string(creds[0].ID),
		SyncStartPolicy: policyFromNow,
		SyncStartDate:   dateFromNow,
	}); err != nil {
		t.Fatalf("AddCompany: %v", err)
	}

	comps, _ := application.Companies.ListCompanies(ctx)
	companyID := comps[0].ID

	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	insertDoc := `
		INSERT INTO documents (
			id, chave_acesso, issue_date, competence, created_at, updated_at,
			prestador_cnpj, prestador_name, tomador_cnpj, tomador_name, intermediario_cnpj, intermediario_name,
			status, layout_version, xml_path, raw_hash, nfse_number, service_description
		)
		VALUES (
			?, ?, ?, ?, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z',
			'45852546000109', 'P', '111', 'T', '', '',
			'normal', '1.0', '', ?, '123', 'Serviço'
		);
	`
	insertRel := `
		INSERT INTO company_documents (
			relation_id, company_id, document_id, company_role, visibility_reason, first_synced_at, last_synced_at
		)
		VALUES (?, ?, ?, 'prestada', 'exact_prestador', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
	`

	_, err := db.ExecContext(ctx, insertDoc, "doc-old", "111", yesterday.Format("2006-01-02T15:04:05Z"), "2026-06", "hash1")
	if err != nil {
		t.Fatalf("insert doc-old err: %v", err)
	}
	_, err = db.ExecContext(ctx, insertRel, "rel-1", string(companyID), "doc-old")
	if err != nil {
		t.Fatalf("insert rel-1 err: %v", err)
	}

	_, err = db.ExecContext(ctx, insertDoc, "doc-new", "222", tomorrow.Format("2006-01-02T15:04:05Z"), "2026-06", "hash2")
	if err != nil {
		t.Fatalf("insert doc-new err: %v", err)
	}
	_, err = db.ExecContext(ctx, insertRel, "rel-2", string(companyID), "doc-new")
	if err != nil {
		t.Fatalf("insert rel-2 err: %v", err)
	}

	// Act
	input := app.ListInput{
		CNPJ: "45852546000109",
	}
	docs, err := application.Documents.ListDocuments(ctx, input)

	if err != nil {
		t.Fatalf("ListDocuments falhou: %v", err)
	}

	// Assert: Only new document should be returned due to from_now policy
	if len(docs) != 1 {
		t.Fatalf("esperava 1 documento (novo), obteve %d", len(docs))
	}
	if docs[0].ID != "doc-new" {
		t.Errorf("esperava ID 'doc-new', obteve '%s'", docs[0].ID)
	}
}

func TestAppIntegration_MarkDocumentsViewed(t *testing.T) {
	application, db := setupTestApp(t)
	ctx := context.Background()

	certPath, _ := filepath.Abs("app_list_integration_test.go")
	if err := application.Credentials.AddCredential(ctx, credential.AddCredentialInput{Label: "L", CertPath: certPath}); err != nil {
		t.Fatalf("AddCredential: %v", err)
	}
	creds, _ := application.Credentials.ListCredentials(ctx)

	if err := application.Companies.AddCompany(ctx, company.AddCompanyInput{
		CNPJ:            "45852546000109",
		Name:            "Company A",
		CredentialID:    string(creds[0].ID),
		Environment:     "producao",
		SyncStartPolicy: "all",
	}); err != nil {
		t.Fatalf("AddCompany: %v", err)
	}

	comp, _ := company.NewStore(db).CompanyByCNPJ(ctx, "45852546000109")

	now := time.Now().Truncate(24 * time.Hour)

	insertDoc := `
		INSERT INTO documents (
			id, chave_acesso, issue_date, competence, created_at, updated_at,
			prestador_cnpj, prestador_name, tomador_cnpj, tomador_name, intermediario_cnpj, intermediario_name,
			status, layout_version, xml_path, raw_hash, nfse_number, service_description
		)
		VALUES (
			?, ?, ?, ?, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z',
			'45852546000109', 'P', '111', 'T', '', '',
			'normal', '1.0', '', ?, '123', 'Serviço'
		);
	`
	insertRel := `
		INSERT INTO company_documents (
			relation_id, company_id, document_id, company_role, visibility_reason, first_synced_at, last_synced_at
		)
		VALUES (?, ?, ?, 'prestada', 'exact_prestador', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
	`

	chaves := map[string]string{
		"doc-1": "35503082245852546000109000000000000126060000000011",
		"doc-2": "35503082245852546000109000000000000226060000000022",
		"doc-3": "35503082245852546000109000000000000326060000000033",
		// A document without chNFSe keeps its infNFSe Id as the chave.
		"doc-4": "NFS35503082245852546000109000000000000426060000000044",
	}
	for _, id := range []string{"doc-1", "doc-2", "doc-3", "doc-4"} {
		if _, err := db.ExecContext(ctx, insertDoc, id, chaves[id], now.Format("2006-01-02T15:04:05Z"), "2026-06", "hash-"+id); err != nil {
			t.Fatalf("insert %s err: %v", id, err)
		}
		if _, err := db.ExecContext(ctx, insertRel, "rel-"+id, string(comp.ID), id); err != nil {
			t.Fatalf("insert rel for %s err: %v", id, err)
		}
	}

	unread := app.ListInput{
		CNPJ:       "45852546000109",
		OnlyUnread: true,
	}

	const cnpj = "45852546000109"
	unreadIDs := func() []string {
		t.Helper()
		docs, err := application.Documents.ListDocuments(ctx, unread)
		if err != nil {
			t.Fatalf("ListDocuments falhou: %v", err)
		}
		var ids []string
		for _, d := range docs {
			ids = append(ids, string(d.ID))
		}
		return ids
	}

	// Marking touches only the given documents.
	count, err := application.Documents.MarkDocumentsViewed(ctx, cnpj, []string{" " + chaves["doc-2"] + " "})
	if err != nil {
		t.Fatalf("MarkDocumentsViewed por chaves falhou: %v", err)
	}
	if count != 1 {
		t.Errorf("esperava marcar 1 documento, marcou %d", count)
	}
	if ids := unreadIDs(); len(ids) != 3 || slices.Contains(ids, "doc-2") {
		t.Errorf("esperava doc-1, doc-3 e doc-4 não lidos, obteve %v", ids)
	}

	// The "NFS" fallback chave is accepted as stored.
	count, err = application.Documents.MarkDocumentsViewed(ctx, cnpj, []string{chaves["doc-4"]})
	if err != nil {
		t.Fatalf("MarkDocumentsViewed com chave NFS falhou: %v", err)
	}
	if count != 1 {
		t.Errorf("esperava marcar o documento da chave NFS, marcou %d", count)
	}

	if _, err := application.Documents.MarkDocumentsViewed(ctx, cnpj, []string{"111"}); err == nil {
		t.Error("esperava erro para chave de acesso inválida")
	}

	// Already-viewed documents are not counted again.
	count, err = application.Documents.MarkDocumentsViewed(ctx, cnpj, []string{chaves["doc-1"], chaves["doc-2"]})
	if err != nil {
		t.Fatalf("MarkDocumentsViewed falhou: %v", err)
	}
	if count != 1 {
		t.Errorf("esperava marcar 1 documento, marcou %d", count)
	}
	if ids := unreadIDs(); !slices.Equal(ids, []string{"doc-3"}) {
		t.Errorf("esperava só doc-3 não lido, obteve %v", ids)
	}
}

func TestAppIntegration_ListEvents(t *testing.T) {
	application, db := setupTestApp(t)
	ctx := context.Background()

	insertDoc := `
		INSERT INTO documents (
			id, chave_acesso, issue_date, competence, created_at, updated_at,
			prestador_cnpj, prestador_name, tomador_cnpj, tomador_name, intermediario_cnpj, intermediario_name,
			status, layout_version, xml_path, raw_hash, nfse_number, service_description
		)
		VALUES (
			'doc-events', '333', '2026-06-20T00:00:00Z', '2026-06', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z',
			'45852546000109', 'P', '111', 'T', '', '',
			'normal', '1.0', '', 'hash3', '123', 'Serviço'
		);
	`
	_, err := db.ExecContext(ctx, insertDoc)
	if err != nil {
		t.Fatalf("insert doc err: %v", err)
	}

	insertEvt := `
		INSERT INTO events (
			id, document_id, chave_acesso, type, event_at, replacement_chave_acesso, description, raw_xml_path, raw_hash, created_at
		)
		VALUES (
			?, 'doc-events', '333', ?, '2026-06-21T00:00:00Z', '', 'desc', '', ?, '2026-01-01T00:00:00Z'
		);
	`
	_, err = db.ExecContext(ctx, insertEvt, "evt-1", "cancelamento", "hash4")
	if err != nil {
		t.Fatalf("insert evt-1 err: %v", err)
	}
	_, err = db.ExecContext(ctx, insertEvt, "evt-2", "substituicao", "hash5")
	if err != nil {
		t.Fatalf("insert evt-2 err: %v", err)
	}

	events, err := application.Documents.ListEventsForDocument(ctx, "doc-events")
	if err != nil {
		t.Fatalf("ListEventsForDocument falhou: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("esperava 2 eventos, obteve %d", len(events))
	}
	if events[0].Type != "cancelamento" {
		t.Errorf("esperava cancelamento, obteve %s", events[0].Type)
	}
}

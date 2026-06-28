package app_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestAppIntegration_ListDocuments(t *testing.T) {
	application, db := setupTestApp(t)
	ctx := context.Background()

	certPath, _ := filepath.Abs("app_list_integration_test.go")
	application.AddCredential(ctx, app.AddCredentialInput{Label: "L", CertPath: certPath})
	creds, _ := application.ListCredentials(ctx)

	now := time.Now().Truncate(24 * time.Hour)
	policyFromNow, dateFromNow, _ := app.ParseSyncStartPolicyInput("from_now", "")

	// Create Company with from_now policy
	application.AddCompany(ctx, app.AddCompanyInput{
		CNPJ:            "45852546000109",
		Name:            "Empresa Listagem",
		Environment:     nfse.EnvironmentRestricted,
		CredentialID:    string(creds[0].ID),
		SyncStartPolicy: policyFromNow,
		SyncStartDate:   dateFromNow,
	})

	comps, _ := application.ListCompanies(ctx)
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
	docs, err := application.ListDocuments(ctx, input)

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
	application.AddCredential(ctx, app.AddCredentialInput{Label: "L", CertPath: certPath})
	creds, _ := application.ListCredentials(ctx)

	application.AddCompany(ctx, app.AddCompanyInput{
		CNPJ:            "45852546000109",
		Name:            "Company A",
		CredentialID:    string(creds[0].ID),
		Environment:     "producao",
		SyncStartPolicy: "all",
	})

	company, _ := application.CompanyRepo.CompanyByCNPJ(ctx, "45852546000109")

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

	_, err := db.ExecContext(ctx, insertDoc, "doc-1", "111", now.Format("2006-01-02T15:04:05Z"), "2026-06", "hash1")
	if err != nil {
		t.Fatalf("insert doc-1 err: %v", err)
	}
	_, err = db.ExecContext(ctx, insertRel, "rel-1", string(company.ID), "doc-1")
	if err != nil {
		t.Fatalf("insert rel-1 err: %v", err)
	}

	_, err = db.ExecContext(ctx, insertDoc, "doc-2", "222", now.Format("2006-01-02T15:04:05Z"), "2026-06", "hash2")
	if err != nil {
		t.Fatalf("insert doc-2 err: %v", err)
	}
	_, err = db.ExecContext(ctx, insertRel, "rel-2", string(company.ID), "doc-2")
	if err != nil {
		t.Fatalf("insert rel-2 err: %v", err)
	}

	input := app.ListInput{
		CNPJ:       "45852546000109",
		OnlyUnread: true,
	}
	count, err := application.MarkDocumentsViewed(ctx, input)
	if err != nil {
		t.Fatalf("MarkDocumentsViewed falhou: %v", err)
	}

	if count != 2 {
		t.Errorf("esperava marcar 2 documentos, marcou %d", count)
	}

	docs, _ := application.ListDocuments(ctx, input)
	if len(docs) != 0 {
		t.Errorf("esperava 0 documentos não lidos, obteve %d", len(docs))
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

	events, err := application.ListEventsForDocument(ctx, "doc-events")
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

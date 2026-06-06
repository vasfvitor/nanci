package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestSQLiteStore_CanonicalAndCompanyDocuments(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	companyA := &nfse.Company{
		ID:           "comp_a",
		CNPJ:         "12345678000199",
		CNPJRoot:     "12345678",
		Name:         "Company A",
		CredentialID: createTestCredential(t, store, "cred-a", "a.pfx", "producao_restrita"),
	}
	companyB := &nfse.Company{
		ID:           "comp_b",
		CNPJ:         "99887766000155",
		CNPJRoot:     "99887766",
		Name:         "Company B",
		CredentialID: createTestCredential(t, store, "cred-b", "b.pfx", "producao_restrita"),
	}

	for _, company := range []*nfse.Company{companyA, companyB} {
		if err := store.CreateCompany(ctx, company); err != nil {
			t.Fatalf("CreateCompany(%s): %v", company.ID, err)
		}
	}

	canonical := &nfse.Document{
		ID:               uuid.NewString(),
		ChaveAcesso:      "NFS123",
		IssueDate:        time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC),
		Competence:       "2026-06",
		PrestadorCNPJ:    companyA.CNPJ,
		TomadorCNPJ:      companyB.CNPJ,
		ServiceValue:     100,
		TotalRetentions:    15.50,
		NFSeNumber:         "9999",
		ServiceDescription: "Teste de descricao truncada",
		Status:           "normal",
		XMLPath:          "xml/2026-06/NFS123.xml",
		RawHash:          "hash1",
	}
	if err := store.UpsertDocument(ctx, canonical); err != nil {
		t.Fatalf("UpsertDocument: %v", err)
	}

	savedCanonical, err := store.GetDocumentByChave(ctx, canonical.ChaveAcesso)
	if err != nil {
		t.Fatalf("GetDocumentByChave: %v", err)
	}
	if savedCanonical == nil {
		t.Fatal("canonical document not found")
	}
	if savedCanonical.NFSeNumber != "9999" {
		t.Errorf("saved NFSeNumber = %q, want 9999", savedCanonical.NFSeNumber)
	}
	if savedCanonical.ServiceDescription != "Teste de descricao truncada" {
		t.Errorf("saved ServiceDescription = %q, want 'Teste de descricao truncada'", savedCanonical.ServiceDescription)
	}
	if savedCanonical.TotalRetentions != 15.50 {
		t.Errorf("saved TotalRetentions = %f, want 15.50", savedCanonical.TotalRetentions)
	}

	relationA := &nfse.CompanyDocument{
		RelationID:        uuid.NewString(),
		CompanyID:         companyA.ID,
		DocumentID:        savedCanonical.ID,
		CompanyRole:       "prestada",
		VisibilityReason:  "exact_prestador",
		FirstSeenNSU:      10,
		LastSeenNSU:       10,
		FirstSeenNSUValid: true,
		LastSeenNSUValid:  true,
	}
	if err := store.UpsertCompanyDocument(ctx, relationA); err != nil {
		t.Fatalf("UpsertCompanyDocument(A): %v", err)
	}

	docsA, err := store.ListDocuments(ctx, companyA.ID, DocumentFilter{})
	if err != nil {
		t.Fatalf("ListDocuments(A): %v", err)
	}
	if len(docsA) != 1 {
		t.Fatalf("expected 1 document for A, got %d", len(docsA))
	}
	if docsA[0].CompanyRole != "prestada" || docsA[0].VisibilityReason != "exact_prestador" {
		t.Fatalf("unexpected company A role/visibility: %+v", docsA[0])
	}

	relationB := &nfse.CompanyDocument{
		RelationID:        uuid.NewString(),
		CompanyID:         companyB.ID,
		DocumentID:        savedCanonical.ID,
		CompanyRole:       "tomada",
		VisibilityReason:  "exact_tomador",
		FirstSeenNSU:      20,
		LastSeenNSU:       20,
		FirstSeenNSUValid: true,
		LastSeenNSUValid:  true,
	}
	if err := store.UpsertCompanyDocument(ctx, relationB); err != nil {
		t.Fatalf("UpsertCompanyDocument(B): %v", err)
	}

	docsAAfter, err := store.ListDocuments(ctx, companyA.ID, DocumentFilter{})
	if err != nil {
		t.Fatalf("ListDocuments(A after): %v", err)
	}
	if len(docsAAfter) != 1 {
		t.Fatalf("expected 1 document for A after B sync, got %d", len(docsAAfter))
	}
	if docsAAfter[0].CompanyRole != "prestada" || docsAAfter[0].VisibilityReason != "exact_prestador" {
		t.Fatalf("company A changed after B sync: %+v", docsAAfter[0])
	}

	docsB, err := store.ListDocuments(ctx, companyB.ID, DocumentFilter{})
	if err != nil {
		t.Fatalf("ListDocuments(B): %v", err)
	}
	if len(docsB) != 1 {
		t.Fatalf("expected 1 document for B, got %d", len(docsB))
	}
	if docsB[0].CompanyRole != "tomada" || docsB[0].VisibilityReason != "exact_tomador" {
		t.Fatalf("unexpected company B role/visibility: %+v", docsB[0])
	}

	var documentsCount, relationsCount int
	if err := store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM documents").Scan(&documentsCount); err != nil {
		t.Fatalf("count documents: %v", err)
	}
	if err := store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM company_documents").Scan(&relationsCount); err != nil {
		t.Fatalf("count company_documents: %v", err)
	}
	if documentsCount != 1 {
		t.Fatalf("expected 1 canonical document, got %d", documentsCount)
	}
	if relationsCount != 2 {
		t.Fatalf("expected 2 company relations, got %d", relationsCount)
	}
}

func TestSQLiteStore_CompanyDocumentIdempotencyAndSameRootVisibility(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	company := &nfse.Company{
		ID:           "comp_root",
		CNPJ:         "12345678000270",
		CNPJRoot:     "12345678",
		Name:         "Branch Company",
		CredentialID: createTestCredential(t, store, "cred-branch", "branch.pfx", "producao_restrita"),
	}
	if err := store.CreateCompany(ctx, company); err != nil {
		t.Fatalf("CreateCompany: %v", err)
	}

	canonical := &nfse.Document{
		ID:            uuid.NewString(),
		ChaveAcesso:   "NFSROOT",
		IssueDate:     time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC),
		Competence:    "2026-06",
		PrestadorCNPJ: "12345678000199",
		TomadorCNPJ:   "99887766000155",
		ServiceValue:  50,
		Status:        "normal",
		XMLPath:       "xml/2026-06/NFSROOT.xml",
		RawHash:       "hash-root",
	}
	if err := store.UpsertDocument(ctx, canonical); err != nil {
		t.Fatalf("UpsertDocument: %v", err)
	}

	savedCanonical, err := store.GetDocumentByChave(ctx, canonical.ChaveAcesso)
	if err != nil {
		t.Fatalf("GetDocumentByChave: %v", err)
	}

	relation := &nfse.CompanyDocument{
		RelationID:        uuid.NewString(),
		CompanyID:         company.ID,
		DocumentID:        savedCanonical.ID,
		CompanyRole:       "none",
		VisibilityReason:  "same_root_only",
		FirstSeenNSU:      30,
		LastSeenNSU:       30,
		FirstSeenNSUValid: true,
		LastSeenNSUValid:  true,
	}
	if err := store.UpsertCompanyDocument(ctx, relation); err != nil {
		t.Fatalf("UpsertCompanyDocument first: %v", err)
	}

	relation.RelationID = uuid.NewString()
	relation.FirstSeenNSU = 30
	relation.LastSeenNSU = 45
	if err := store.UpsertCompanyDocument(ctx, relation); err != nil {
		t.Fatalf("UpsertCompanyDocument second: %v", err)
	}

	docs, err := store.ListDocuments(ctx, company.ID, DocumentFilter{})
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}
	if docs[0].CompanyRole != "none" || docs[0].VisibilityReason != "same_root_only" {
		t.Fatalf("unexpected role/visibility: %+v", docs[0])
	}
	if !docs[0].FirstSeenNSUValid || docs[0].FirstSeenNSU != 30 {
		t.Fatalf("unexpected first NSU: %+v", docs[0])
	}
	if !docs[0].LastSeenNSUValid || docs[0].LastSeenNSU != 45 {
		t.Fatalf("unexpected last NSU: %+v", docs[0])
	}

	var relationsCount int
	if err := store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM company_documents").Scan(&relationsCount); err != nil {
		t.Fatalf("count company_documents: %v", err)
	}
	if relationsCount != 1 {
		t.Fatalf("expected 1 company relation after idempotent upsert, got %d", relationsCount)
	}
}

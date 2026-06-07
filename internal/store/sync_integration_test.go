package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

func TestCompanyCredentialPersistenceAndAssignment(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	credentials := NewCredentialRepository(db)
	companies := NewCompanyRepository(db)

	first := testCredential("credential-1", nfse.EnvironmentRestricted)
	if err := credentials.CreateCredential(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	company := testCompany("company-1", "11222333000181", first)
	if err := companies.CreateCompany(context.Background(), company); err != nil {
		t.Fatal(err)
	}

	stored, err := companies.CompanyByCNPJ(context.Background(), company.CNPJ)
	if err != nil {
		t.Fatal(err)
	}
	assertCompanyCredential(t, stored, first)

	second := testCredential("credential-2", nfse.EnvironmentProduction)
	second.Label = "Production certificate"
	second.CertPath = `C:\certs\production.pfx`
	if err := credentials.CreateCredential(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	if err := companies.AssignCredential(context.Background(), company.ID, second.ID); err != nil {
		t.Fatal(err)
	}

	stored, err = companies.CompanyByCNPJ(context.Background(), company.CNPJ)
	if err != nil {
		t.Fatal(err)
	}
	assertCompanyCredential(t, stored, second)
}

func TestDocumentUpsertUsesCanonicalIDAndListsRelations(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	syncRepo := NewSyncRepository(db)
	documentRepo := NewDocumentRepository(db)
	firstCompany := seedCompany(t, db, "company-1", "11222333000181")
	secondCompany := seedCompany(t, db, "company-2", "11222333000262")

	doc := testDocument("document-1", "12345678901234567890123456789012345678901234567890", "hash-1")
	applyDocument(t, syncRepo, firstCompany.ID, doc, 10)

	repeated := doc
	repeated.ID = "document-2"
	repeated.RawHash = "hash-2"
	repeated.ServiceDescription = "updated"
	applyDocument(t, syncRepo, firstCompany.ID, repeated, 20)
	applyDocument(t, syncRepo, secondCompany.ID, repeated, 30)

	var documentCount int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM documents WHERE chave_acesso = ?`, doc.ChaveAcesso).Scan(&documentCount); err != nil {
		t.Fatal(err)
	}
	if documentCount != 1 {
		t.Fatalf("document count = %d, want 1", documentCount)
	}

	var relationCount int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM company_documents WHERE document_id = ?`, doc.ID).Scan(&relationCount); err != nil {
		t.Fatal(err)
	}
	if relationCount != 2 {
		t.Fatalf("relation count = %d, want 2", relationCount)
	}

	limit := 1
	fromNSU := int64(15)
	docs, err := documentRepo.ListCompanyDocuments(context.Background(), firstCompany.ID, nfse.DocumentFilter{
		Competence: "2026-06",
		Direction:  string(nfse.CompanyRoleTomada),
		Status:     string(nfse.DocumentStatusNormal),
		FromNSU:    &fromNSU,
		Limit:      &limit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("listed documents = %d, want 1", len(docs))
	}
	got := docs[0]
	if got.ID != doc.ID || got.DocumentID != doc.ID {
		t.Fatalf("canonical IDs = %q/%q, want %q", got.ID, got.DocumentID, doc.ID)
	}
	if got.RelationID == "" {
		t.Fatal("relation ID was not loaded")
	}
	if got.FirstSeenNSU != 10 || got.LastSeenNSU != 20 {
		t.Fatalf("NSU range = %d-%d, want 10-20", got.FirstSeenNSU, got.LastSeenNSU)
	}
	if !got.FirstSeenNSUValid || !got.LastSeenNSUValid {
		t.Fatal("NSU validity flags were not loaded")
	}
}

func TestEventsUpdateStatusAndLinkWhenDocumentArrives(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	syncRepo := NewSyncRepository(db)
	documentRepo := NewDocumentRepository(db)
	company := seedCompany(t, db, "company-1", "11222333000181")
	accessKey := nfse.AccessKey("12345678901234567890123456789012345678901234567890")

	applyEvent(t, syncRepo, nfse.Event{
		ID:          "event-1",
		ChaveAcesso: accessKey,
		Type:        "cancelamento",
		RawHash:     "event-hash-1",
		RawXMLPath:  "event-hash-1.xml",
		Description: "cancelled",
	}, company.ID, 5)

	doc := testDocument("document-1", string(accessKey), "document-hash")
	applyDocument(t, syncRepo, company.ID, doc, 10)
	assertDocumentStatus(t, db, accessKey, nfse.DocumentStatusCancelada)

	var linkedDocumentID sql.NullString
	if err := db.QueryRowContext(context.Background(), `SELECT document_id FROM events WHERE id = ?`, "event-1").Scan(&linkedDocumentID); err != nil {
		t.Fatal(err)
	}
	if !linkedDocumentID.Valid || linkedDocumentID.String != string(doc.ID) {
		t.Fatalf("linked document ID = %#v, want %q", linkedDocumentID, doc.ID)
	}

	substitution := nfse.Event{
		ID:                     "event-2",
		ChaveAcesso:            accessKey,
		Type:                   "substituicao",
		ReplacementChaveAcesso: "09876543210987654321098765432109876543210987654321",
		RawHash:                "event-hash-2",
		RawXMLPath:             "event-hash-2.xml",
		Description:            "replaced",
	}
	applyEvent(t, syncRepo, substitution, company.ID, 11)
	applyEvent(t, syncRepo, substitution, company.ID, 11)
	assertDocumentStatus(t, db, accessKey, nfse.DocumentStatusSubstituida)

	events, err := documentRepo.ListEventsByDocument(context.Background(), string(doc.ID))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2 after duplicate hash", len(events))
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	return db
}

func testCredential(id string, environment nfse.Environment) *nfse.Credential {
	return &nfse.Credential{
		ID:            nfse.CredentialID(id),
		Label:         "Certificate",
		CertPath:      `C:\certs\company.pfx`,
		Environment:   environment,
		OwnerCNPJ:     "11222333000181",
		OwnerCNPJRoot: "11222333",
	}
}

func testCompany(id, cnpj string, credential *nfse.Credential) *nfse.Company {
	return &nfse.Company{
		ID:                 nfse.CompanyID(id),
		CNPJ:               cnpj,
		CNPJRoot:           cnpj[:8],
		Name:               id,
		CredentialID:       credential.ID,
		CredentialLabel:    credential.Label,
		CredentialCertPath: credential.CertPath,
		Environment:        credential.Environment,
	}
}

func seedCompany(t *testing.T, db *sql.DB, id, cnpj string) *nfse.Company {
	t.Helper()

	credential := testCredential("credential-"+id, nfse.EnvironmentRestricted)
	if err := NewCredentialRepository(db).CreateCredential(context.Background(), credential); err != nil {
		t.Fatal(err)
	}
	company := testCompany(id, cnpj, credential)
	if err := NewCompanyRepository(db).CreateCompany(context.Background(), company); err != nil {
		t.Fatal(err)
	}
	return company
}

func testDocument(id, accessKey, hash string) nfse.Document {
	return nfse.Document{
		ID:                 nfse.DocumentID(id),
		ChaveAcesso:        nfse.AccessKey(accessKey),
		IssueDate:          time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC),
		Competence:         "2026-06",
		PrestadorCNPJ:      "99887766000155",
		PrestadorName:      "Provider",
		TomadorCNPJ:        "11222333000181",
		TomadorName:        "Customer",
		ServiceValue:       10000,
		Status:             nfse.DocumentStatusNormal,
		LayoutVersion:      "1.01",
		XMLPath:            hash + ".xml",
		RawHash:            hash,
		NFSeNumber:         "123",
		ServiceDescription: "service",
	}
}

func applyDocument(t *testing.T, repo *SyncRepository, companyID nfse.CompanyID, document nfse.Document, nsu int64) {
	t.Helper()
	err := repo.ApplyDocument(context.Background(), nfse.ApplyDocumentParams{
		Document: document,
		Participation: nfse.CompanyParticipation{
			CompanyRole:      nfse.CompanyRoleTomada,
			VisibilityReason: "exact_tomador",
		},
		CompanyID: companyID,
		NSU:       nsu,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func applyEvent(t *testing.T, repo *SyncRepository, event nfse.Event, companyID nfse.CompanyID, nsu int64) {
	t.Helper()
	if err := repo.ApplyEvent(context.Background(), nfse.ApplyEventParams{
		Event:     event,
		CompanyID: companyID,
		NSU:       nsu,
	}); err != nil {
		t.Fatal(err)
	}
}

func assertCompanyCredential(t *testing.T, company *nfse.Company, credential *nfse.Credential) {
	t.Helper()
	if company.CredentialID != credential.ID ||
		company.CredentialLabel != credential.Label ||
		company.CredentialCertPath != credential.CertPath ||
		company.Environment != credential.Environment {
		t.Fatalf("company credential metadata = %#v, want %#v", company, credential)
	}
}

func assertDocumentStatus(t *testing.T, db *sql.DB, accessKey nfse.AccessKey, want nfse.DocumentStatus) {
	t.Helper()
	var got string
	if err := db.QueryRowContext(context.Background(), `SELECT status FROM documents WHERE chave_acesso = ?`, accessKey).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("status = %q, want %q", got, want)
	}
}

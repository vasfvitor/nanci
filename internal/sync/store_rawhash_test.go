package sync_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store/storetest"
	"github.com/vasfvitor/nanci/internal/sync"
)

// prefixedChaveDigits is a NFS-e chave whose XML has no chNFSe: builds older
// than the parser fix stored it as "NFS" + these digits.
const prefixedChaveDigits = "26079012245852546000109000000000003126084764398700"

func applyNFSe(t *testing.T, repo *sync.Store, companyID dfe.CompanyID, doc nfse.Document) nfse.ApplyOutcome {
	t.Helper()
	outcome, err := repo.ApplyDocument(context.Background(), nfse.ApplyDocumentParams{
		Document: doc,
		Participation: nfse.CompanyParticipation{
			CompanyRole:      nfse.CompanyRoleTomada,
			VisibilityReason: "exact_tomador",
		},
		CompanyID: companyID,
		NSU:       31,
	})
	if err != nil {
		t.Fatalf("apply %s: %v", doc.ID, err)
	}
	return outcome
}

func documentByID(t *testing.T, db *sql.DB, id string) (chave, rawHash string) {
	t.Helper()
	if err := db.QueryRowContext(context.Background(),
		`SELECT chave_acesso, raw_hash FROM documents WHERE id = ?`, id,
	).Scan(&chave, &rawHash); err != nil {
		t.Fatalf("read document %s: %v", id, err)
	}
	return chave, rawHash
}

func linkCount(t *testing.T, db *sql.DB, companyID dfe.CompanyID, documentID string) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM company_documents WHERE company_id = ? AND document_id = ?`,
		string(companyID), documentID,
	).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestApplyDocumentRenamesTheRowWithTheSameXML: syncing again an NSU stored
// with the "NFS" prefix renames that row instead of failing on
// UNIQUE(documents.raw_hash).
func TestApplyDocumentRenamesTheRowWithTheSameXML(t *testing.T) {
	t.Parallel()
	db := storetest.OpenTestDB(t)
	repo := sync.NewStore(db)
	company := seedCompany(t, db, "company-1", "11222333000181")

	applyNFSe(t, repo, company.ID, testDocument("doc-old", "NFS"+prefixedChaveDigits, "hash-same-xml"))
	outcome := applyNFSe(t, repo, company.ID, testDocument("doc-new", prefixedChaveDigits, "hash-same-xml"))

	if outcome.Inserted {
		t.Error("Inserted = true, want false: the company already had the document")
	}
	var count int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM documents`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("documents = %d, want 1", count)
	}
	if chave, rawHash := documentByID(t, db, "doc-old"); chave != prefixedChaveDigits || rawHash != "hash-same-xml" {
		t.Errorf("doc-old = (%q, %q), want the 50 digits and the same raw_hash", chave, rawHash)
	}
	if n := linkCount(t, db, company.ID, "doc-old"); n != 1 {
		t.Errorf("company links to doc-old = %d, want 1", n)
	}
}

// TestApplyDocumentLinksTheTwinOfAPrefixedRow: when another document already
// has the 50 digits, the prefixed row stays as it is (as migration 020 left
// it) and the company is linked to the document with the digits.
func TestApplyDocumentLinksTheTwinOfAPrefixedRow(t *testing.T) {
	t.Parallel()
	db := storetest.OpenTestDB(t)
	repo := sync.NewStore(db)
	first := seedCompany(t, db, "company-1", "11222333000181")
	second := seedCompany(t, db, "company-2", "11444777000161")

	applyNFSe(t, repo, first.ID, testDocument("doc-old", "NFS"+prefixedChaveDigits, "hash-old-xml"))
	applyNFSe(t, repo, first.ID, testDocument("doc-twin", prefixedChaveDigits, "hash-twin-xml"))

	outcome := applyNFSe(t, repo, second.ID, testDocument("doc-new", prefixedChaveDigits, "hash-old-xml"))

	if !outcome.Inserted {
		t.Error("Inserted = false, want true: the second company had no link yet")
	}
	if chave, rawHash := documentByID(t, db, "doc-old"); chave != "NFS"+prefixedChaveDigits || rawHash != "hash-old-xml" {
		t.Errorf("doc-old = (%q, %q), want it untouched", chave, rawHash)
	}
	if chave, rawHash := documentByID(t, db, "doc-twin"); chave != prefixedChaveDigits || rawHash != "hash-twin-xml" {
		t.Errorf("doc-twin = (%q, %q), want it untouched", chave, rawHash)
	}
	if n := linkCount(t, db, second.ID, "doc-twin"); n != 1 {
		t.Errorf("second company links to doc-twin = %d, want 1", n)
	}
}

package app

import (
	"context"
	"testing"

	"github.com/vasfvitor/nanci/internal/cte"
)

func (e *nfeTestEnv) importCTe(paths ...string) ImportXMLResult {
	e.t.Helper()
	result, err := e.app.CTe.ImportXML(context.Background(), ImportXMLInput{CNPJ: nfeTestCNPJ, Files: paths})
	if err != nil {
		e.t.Fatalf("ImportXML: %v", err)
	}
	if len(result.Files) != len(paths) {
		e.t.Fatalf("files = %d, want %d", len(result.Files), len(paths))
	}
	return result
}

func cteFixtureFile(t *testing.T, fixture string) string {
	t.Helper()
	return writeImportFile(t, fixture, readCTeFixture(t, fixture))
}

func TestCTeImportXML(t *testing.T) {
	env := newNFeTestEnv(t)
	ctx := context.Background()
	cancelamento := cteFixtureFile(t, "proceventocte-cancelamento.xml")

	alone := env.importCTe(cancelamento)
	checkRejected(t, alone.Files[0], "CT-e desconhecido para a empresa")

	result := env.importCTe(cteFixtureFile(t, "procte.xml"), cteFixtureFile(t, "procteos.xml"), cancelamento, nfeFixtureFile(t, "procnfe.xml"))
	checkImported(t, result.Files[0], ImportKindCompleta, cteChaveProc, ImportOutcomeImported)
	checkImported(t, result.Files[1], ImportKindCompleta, cteChaveOS, ImportOutcomeImported)
	checkImported(t, result.Files[2], ImportKindEvento, cteChaveProc, ImportOutcomeImported)
	checkRejected(t, result.Files[3], "não é um XML de CT-e: raiz <nfeProc>")
	if result.Imported != 3 || result.Updated != 0 || result.Rejected != 1 {
		t.Errorf("counts = %d/%d/%d, want 3 imported and 1 rejected", result.Imported, result.Updated, result.Rejected)
	}

	docs, err := env.app.CTe.ListDocuments(ctx, ListCTeInput{CNPJ: nfeTestCNPJ, ChavesAcesso: []string{cteChaveProc}})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || docs[0].CompanyRole != cte.CompanyRoleTomador || docs[0].Situacao != cte.SituacaoCancelada {
		t.Fatalf("procte.xml = %+v, want the tomador's cancelada CT-e", docs)
	}
	var nsu *int64
	if err := env.db.QueryRowContext(context.Background(), `SELECT first_seen_nsu FROM company_cte_documents WHERE company_id = ? LIMIT 1`, string(env.company.ID)).Scan(&nsu); err != nil {
		t.Fatal(err)
	}
	if nsu != nil {
		t.Errorf("first_seen_nsu = %d, want NULL", *nsu)
	}

	again := env.importCTe(cteFixtureFile(t, "procte.xml"))
	checkImported(t, again.Files[0], ImportKindCompleta, cteChaveProc, ImportOutcomeUpdated)
}

func TestCTeImportXMLRejects(t *testing.T) {
	env := newNFeTestEnv(t)

	// procte-v200-toma03.xml is a homologação CT-e; the company is in produção.
	result := env.importCTe(cteFixtureFile(t, "procte-v200-toma03.xml"), cteFixtureFile(t, "invalid.xml"))
	checkRejected(t, result.Files[0], "XML de homologação em empresa de produção")
	checkRejected(t, result.Files[1], "XML inválido")
	if result.Rejected != 2 {
		t.Errorf("rejected = %d, want 2", result.Rejected)
	}
}

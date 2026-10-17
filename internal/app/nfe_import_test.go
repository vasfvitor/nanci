package app

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/nfe"
)

// writeImportFile writes the content to a file named name in a temp dir and
// returns its path.
func writeImportFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// nfeFixtureFile writes the NF-e fixture, after the old, new replacements,
// to a temp file and returns its path.
func nfeFixtureFile(t *testing.T, fixture string, replacements ...string) string {
	t.Helper()
	return writeImportFile(t, fixture, strings.NewReplacer(replacements...).Replace(readNFeFixture(t, fixture)))
}

func (e *nfeTestEnv) importNFe(paths ...string) ImportXMLResult {
	e.t.Helper()
	result, err := e.app.NFe.ImportXML(context.Background(), ImportXMLInput{CNPJ: nfeTestCNPJ, Files: paths})
	if err != nil {
		e.t.Fatalf("ImportXML: %v", err)
	}
	if len(result.Files) != len(paths) {
		e.t.Fatalf("files = %d, want %d", len(result.Files), len(paths))
	}
	return result
}

func (e *nfeTestEnv) nfeDocument(chave string) NFeDocument {
	e.t.Helper()
	docs, err := e.app.NFe.ListDocuments(context.Background(), NFeListInput{CNPJ: nfeTestCNPJ, ChavesAcesso: []string{chave}})
	if err != nil {
		e.t.Fatal(err)
	}
	if len(docs) != 1 {
		e.t.Fatalf("documents for %s = %d, want 1", chave, len(docs))
	}
	return docs[0]
}

func checkImported(t *testing.T, got ImportedFile, kind, chave, outcome string) {
	t.Helper()
	if got.Kind != kind || got.ChaveAcesso != chave || got.Outcome != outcome {
		t.Errorf("file = %+v, want %s %s %s", got, kind, chave, outcome)
	}
}

func checkRejected(t *testing.T, got ImportedFile, message string) {
	t.Helper()
	if got.Outcome != ImportOutcomeRejected || !strings.Contains(got.Message, message) {
		t.Errorf("file = %+v, want rejected with %q", got, message)
	}
}

func TestNFeImportXMLProcNFe(t *testing.T) {
	env := newNFeTestEnv(t)
	path := nfeFixtureFile(t, "procnfe.xml")

	result := env.importNFe(path)
	checkImported(t, result.Files[0], ImportKindCompleta, nfeChaveProc, ImportOutcomeImported)
	if result.Files[0].Path != path || result.Imported != 1 || result.Updated != 0 || result.Rejected != 0 {
		t.Errorf("result = %+v, want one imported file at %s", result, path)
	}

	doc := env.nfeDocument(nfeChaveProc)
	if doc.CompanyRole != nfe.CompanyRoleDestinatario || doc.Completeness != nfe.CompletenessCompleta || doc.TpAmb != "1" {
		t.Errorf("document = %s %s tpAmb %s, want destinatario completa 1", doc.CompanyRole, doc.Completeness, doc.TpAmb)
	}
	if _, err := env.xml.Get(doc.RawHash); err != nil {
		t.Errorf("stored XML %s: %v", doc.RawHash, err)
	}
	var firstNSU, lastNSU sql.NullInt64
	if err := env.db.QueryRowContext(context.Background(), `SELECT first_seen_nsu, last_seen_nsu FROM company_nfe_documents WHERE company_id = ?`, string(env.company.ID)).Scan(&firstNSU, &lastNSU); err != nil {
		t.Fatal(err)
	}
	if firstNSU.Valid || lastNSU.Valid {
		t.Errorf("NSU = %v/%v, want NULL for an imported XML", firstNSU, lastNSU)
	}

	again := env.importNFe(path)
	checkImported(t, again.Files[0], ImportKindCompleta, nfeChaveProc, ImportOutcomeUpdated)
	if again.Imported != 0 || again.Updated != 1 {
		t.Errorf("second import = %+v, want one updated file", again)
	}
}

func TestNFeImportXMLResumoThenCompleta(t *testing.T) {
	env := newNFeTestEnv(t)

	resumo := env.importNFe(nfeFixtureFile(t, "resnfe.xml"))
	checkImported(t, resumo.Files[0], ImportKindResumo, nfeChaveProc, ImportOutcomeImported)
	if doc := env.nfeDocument(nfeChaveProc); doc.Completeness != nfe.CompletenessResumo {
		t.Errorf("after resNFe: completeness = %s, want resumo", doc.Completeness)
	}

	completa := env.importNFe(nfeFixtureFile(t, "procnfe.xml"))
	checkImported(t, completa.Files[0], ImportKindCompleta, nfeChaveProc, ImportOutcomeUpdated)
	if doc := env.nfeDocument(nfeChaveProc); doc.Completeness != nfe.CompletenessCompleta {
		t.Errorf("after procNFe: completeness = %s, want completa", doc.Completeness)
	}
}

func TestNFeImportXMLEvents(t *testing.T) {
	env := newNFeTestEnv(t)
	cancelamento := nfeFixtureFile(t, "proceventonfe-cancelamento.xml")

	// The cancelamento is for an NF-e the company does not hold and was
	// authored by the emitente.
	alone := env.importNFe(cancelamento)
	checkRejected(t, alone.Files[0], "NF-e desconhecida para a empresa")
	if alone.Files[0].Kind != ImportKindEvento || alone.Files[0].ChaveAcesso != nfeChaveCancelada {
		t.Errorf("rejected event = %+v, want its kind and chave", alone.Files[0])
	}

	nota := nfeFixtureFile(t, "procnfe.xml", nfeChaveProc, nfeChaveCancelada)
	result := env.importNFe(nota, cancelamento)
	checkImported(t, result.Files[0], ImportKindCompleta, nfeChaveCancelada, ImportOutcomeImported)
	checkImported(t, result.Files[1], ImportKindEvento, nfeChaveCancelada, ImportOutcomeImported)
	if result.Imported != 2 || result.Rejected != 0 {
		t.Errorf("result = %+v, want two imported files", result)
	}
	if doc := env.nfeDocument(nfeChaveCancelada); doc.Situacao != nfe.SituacaoCancelada {
		t.Errorf("situação = %s, want cancelada", doc.Situacao)
	}

	again := env.importNFe(cancelamento)
	checkImported(t, again.Files[0], ImportKindEvento, nfeChaveCancelada, ImportOutcomeUpdated)
}

// TestNFeImportXMLOwnEventWithoutDocument keeps an event the company
// authored, as the distribution does, even before the NF-e arrives.
func TestNFeImportXMLOwnEventWithoutDocument(t *testing.T) {
	env := newNFeTestEnv(t)
	result := env.importNFe(nfeFixtureFile(t, "proceventonfe-ciencia.xml"))
	checkImported(t, result.Files[0], ImportKindEvento, nfeChaveProc, ImportOutcomeImported)
}

func TestNFeImportXMLRejects(t *testing.T) {
	env := newNFeTestEnv(t)
	dir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		message string
	}{
		{"homologação", nfeFixtureFile(t, "procnfe.xml", "<tpAmb>1</tpAmb>", "<tpAmb>2</tpAmb>"), "XML de homologação em empresa de produção"},
		{"not a party", nfeFixtureFile(t, "procnfe.xml", nfeTestCNPJ, "13579246000101"), "a empresa 70.860.312/0001-50 não é parte desta NF-e"},
		{"invalid", nfeFixtureFile(t, "invalid.xml"), "XML inválido"},
		{"other root", writeImportFile(t, "cte.xml", readCTeFixture(t, "procte.xml")), "não é um XML de NF-e: raiz <cteProc>"},
		{"parse error", writeImportFile(t, "broken.xml", `<nfeProc><NFe/></nfeProc>`), ""},
		{"missing", filepath.Join(dir, "nada.xml"), "ler arquivo"},
		{"too large", writeImportFile(t, "big.xml", strings.Repeat(" ", importMaxBytes+1)), "maior que 20 MiB"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := env.importNFe(tc.path)
			checkRejected(t, result.Files[0], tc.message)
			if result.Files[0].Message == "" || result.Rejected != 1 {
				t.Errorf("result = %+v, want one rejection with a message", result)
			}
		})
	}

	docs, err := env.app.NFe.ListDocuments(context.Background(), NFeListInput{CNPJ: nfeTestCNPJ})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 0 {
		t.Errorf("documents after rejections = %d, want 0", len(docs))
	}
}

func TestNFeImportXMLCounts(t *testing.T) {
	env := newNFeTestEnv(t)
	result := env.importNFe(nfeFixtureFile(t, "procnfe.xml"), nfeFixtureFile(t, "invalid.xml"), nfeFixtureFile(t, "procnfe-denegada.xml"))
	if result.Imported != 2 || result.Updated != 0 || result.Rejected != 1 {
		t.Errorf("counts = %d/%d/%d, want 2 imported and 1 rejected", result.Imported, result.Updated, result.Rejected)
	}
	if result.Files[1].Outcome != ImportOutcomeRejected || result.Files[2].ChaveAcesso != nfeChaveDenegada {
		t.Errorf("files = %+v, want the input order", result.Files)
	}

	if _, err := env.app.NFe.ImportXML(context.Background(), ImportXMLInput{CNPJ: "11222333000181"}); err == nil {
		t.Error("ImportXML of an unknown company: want an error")
	}
}

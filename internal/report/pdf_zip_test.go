package report

import (
	"archive/zip"
	"errors"
	"io"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// fakeRender stands in for a DANFE/DACTE/DANFSe renderer: the "PDF" names
// the XML it came from and the watermark flag.
func fakeRender(xml []byte, cancelada bool) ([]byte, error) {
	return []byte("pdf(" + string(xml) + "," + strconv.FormatBool(cancelada) + ")"), nil
}

func TestNFSePDFEntries(t *testing.T) {
	got := NFSePDFEntries([]nfse.CompanyDocument{
		{Document: nfse.Document{ChaveAcesso: "A", Competence: "2026-06", RawHash: "ha", Status: nfse.DocumentStatusNormal}, CompanyRole: nfse.CompanyRolePrestada},
		{Document: nfse.Document{ChaveAcesso: "B", Competence: "2026-06", RawHash: "hb", Status: nfse.DocumentStatusCancelada}, CompanyRole: nfse.CompanyRoleTomada},
		{Document: nfse.Document{ChaveAcesso: "C", Competence: "2026-06"}, CompanyRole: nfse.CompanyRoleTomada},
		{Document: nfse.Document{ChaveAcesso: "D", RawHash: "hd", Status: nfse.DocumentStatusSubstituida}, CompanyRole: ""},
	})
	want := []ZipEntry{
		{Path: "2026-06/prestada/A.pdf", RawHash: "ha"},
		{Path: "2026-06/tomada/B.pdf", RawHash: "hb", Cancelada: true},
		{Path: "sem-papel-fiscal/D.pdf", RawHash: "hd"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("entries = %+v, want %+v", got, want)
	}
}

func TestNFePDFEntriesSkipsResumo(t *testing.T) {
	got := NFePDFEntries([]nfe.CompanyDocument{
		{Document: nfe.Document{ChaveAcesso: "A", Competence: "2026-06", RawHash: "ha", Completeness: nfe.CompletenessCompleta, Situacao: nfe.SituacaoAutorizada}, CompanyRole: nfe.CompanyRoleDestinatario},
		{Document: nfe.Document{ChaveAcesso: "B", Competence: "2026-06", RawHash: "hb", Completeness: nfe.CompletenessCompleta, Situacao: nfe.SituacaoCancelada}, CompanyRole: nfe.CompanyRoleDestinatario},
		{Document: nfe.Document{ChaveAcesso: "C", Competence: "2026-06", RawHash: "hc", Completeness: nfe.CompletenessResumo, Situacao: nfe.SituacaoAutorizada}, CompanyRole: nfe.CompanyRoleDestinatario},
		{Document: nfe.Document{ChaveAcesso: "D", Competence: "2026-06", Completeness: nfe.CompletenessCompleta}, CompanyRole: nfe.CompanyRoleDestinatario},
	})
	want := []ZipEntry{
		{Path: "2026-06/destinatario/A.pdf", RawHash: "ha"},
		{Path: "2026-06/destinatario/B.pdf", RawHash: "hb", Cancelada: true},
	}
	if !slices.Equal(got, want) {
		t.Errorf("entries = %+v, want %+v", got, want)
	}
}

func TestCTePDFEntriesKeepsOnlyCTe(t *testing.T) {
	doc := func(chave, hash string, tipo cte.TipoDocumento, situacao cte.Situacao) cte.CompanyDocument {
		return cte.CompanyDocument{
			Document:    cte.Document{ChaveAcesso: dfe.AccessKey(chave), Competence: "2026-06", RawHash: hash, TipoDocumento: tipo, Situacao: situacao},
			CompanyRole: cte.CompanyRoleTomador,
		}
	}
	got := CTePDFEntries([]cte.CompanyDocument{
		doc("A", "ha", cte.TipoDocumentoCTe, cte.SituacaoAutorizada),
		doc("B", "hb", cte.TipoDocumentoCTe, cte.SituacaoCancelada),
		doc("C", "hc", cte.TipoDocumentoCTeOS, cte.SituacaoAutorizada),
		doc("D", "hd", cte.TipoDocumentoGTVe, cte.SituacaoAutorizada),
		doc("E", "he", cte.TipoDocumentoCTeSimplificado, cte.SituacaoAutorizada),
		doc("F", "", cte.TipoDocumentoCTe, cte.SituacaoAutorizada),
	})
	want := []ZipEntry{
		{Path: "2026-06/tomador/A.pdf", RawHash: "ha"},
		{Path: "2026-06/tomador/B.pdf", RawHash: "hb", Cancelada: true},
	}
	if !slices.Equal(got, want) {
		t.Errorf("entries = %+v, want %+v", got, want)
	}
}

func TestGeneratePDFZIPWritesRenderedEntries(t *testing.T) {
	dir := t.TempDir()
	store := files.NewBlobStore(dir)
	for hash, xml := range map[string]string{"ha": "<a/>", "hb": "<b/>"} {
		if err := store.Store(hash, []byte(xml)); err != nil {
			t.Fatalf("Store: %v", err)
		}
	}
	entries := []ZipEntry{
		{Path: "2026-06/prestada/A.pdf", RawHash: "ha"},
		{Path: "2026-06/tomada/B.pdf", RawHash: "hb", Cancelada: true},
	}
	outPath := filepath.Join(dir, "pdfs.zip")
	if err := GeneratePDFZIP(entries, store, fakeRender, outPath); err != nil {
		t.Fatalf("GeneratePDFZIP: %v", err)
	}

	got := readZIP(t, outPath)
	want := map[string]string{
		"2026-06/prestada/A.pdf": "pdf(<a/>,false)",
		"2026-06/tomada/B.pdf":   "pdf(<b/>,true)",
	}
	if len(got) != len(want) {
		t.Fatalf("archive = %v, want %v", got, want)
	}
	for name, content := range want {
		if got[name] != content {
			t.Errorf("%s = %q, want %q", name, got[name], content)
		}
	}
}

func TestGeneratePDFZIPRenderErrorNamesEntry(t *testing.T) {
	dir := t.TempDir()
	store := files.NewBlobStore(dir)
	if err := store.Store("ha", []byte("<a/>")); err != nil {
		t.Fatalf("Store: %v", err)
	}
	errBroken := errors.New("broken layout")
	render := func([]byte, bool) ([]byte, error) { return nil, errBroken }

	err := GeneratePDFZIP([]ZipEntry{{Path: "2026-06/prestada/A.pdf", RawHash: "ha"}}, store, render, filepath.Join(dir, "pdfs.zip"))
	if !errors.Is(err, errBroken) {
		t.Fatalf("error = %v, want %v", err, errBroken)
	}
	if !strings.Contains(err.Error(), "2026-06/prestada/A.pdf") {
		t.Errorf("error %q does not name the entry", err)
	}
}

func TestGeneratePDFZIPFailsWhenBlobIsMissing(t *testing.T) {
	dir := t.TempDir()
	store := files.NewBlobStore(dir)

	err := GeneratePDFZIP([]ZipEntry{{Path: "A.pdf", RawHash: "missing"}}, store, fakeRender, filepath.Join(dir, "pdfs.zip"))
	if !errors.Is(err, files.ErrBlobNotFound) {
		t.Fatalf("error = %v, want ErrBlobNotFound", err)
	}
}

func readZIP(t *testing.T, path string) map[string]string {
	t.Helper()
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	defer func() { _ = archive.Close() }()

	out := map[string]string{}
	for _, f := range archive.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		out[f.Name] = string(data)
	}
	return out
}

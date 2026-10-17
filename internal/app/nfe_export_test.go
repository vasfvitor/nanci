package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/vasfvitor/nanci/internal/danfe"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/sefaz"
)

// fakeDANFE records each render and returns a fixed PDF, or err.
type fakeDANFE struct {
	calls []fakeDANFECall
	err   error
}

type fakeDANFECall struct {
	xml  []byte
	opts danfe.Options
}

func (f *fakeDANFE) Render(xml []byte, opts danfe.Options) ([]byte, error) {
	f.calls = append(f.calls, fakeDANFECall{xml: xml, opts: opts})
	if f.err != nil {
		return nil, f.err
	}
	return []byte("%PDF-fake"), nil
}

// cancelledFor returns the Cancelada option of the render whose XML carries
// chave.
func (f *fakeDANFE) cancelledFor(t *testing.T, chave string) bool {
	t.Helper()
	for _, c := range f.calls {
		if bytes.Contains(c.xml, []byte(chave)) {
			return c.opts.Cancelada
		}
	}
	t.Fatalf("no render for %s", chave)
	return false
}

func (e *nfeTestEnv) useFakeDANFE() *fakeDANFE {
	fake := &fakeDANFE{}
	e.app.NFe.DANFERenderer = fake
	return fake
}

// pendingChaves returns the sorted chaves not yet exported with kind.
func (e *nfeTestEnv) pendingChaves(kind string) []string {
	e.t.Helper()
	docs, err := e.repo.ListPendingExport(context.Background(), e.company.ID, nfe.DocumentFilter{TpAmb: sefaz.TpAmbProducao}, kind)
	if err != nil {
		e.t.Fatal(err)
	}
	chaves := make([]string, 0, len(docs))
	for _, d := range docs {
		chaves = append(chaves, string(d.ChaveAcesso))
	}
	slices.Sort(chaves)
	return chaves
}

// seedCancelledCompleta seeds the fixtures and cancels the completa
// nfeChaveProc with the cancelamento event retargeted at its chave.
func (e *nfeTestEnv) seedCancelledCompleta() {
	e.t.Helper()
	e.seedFixtures()
	e.seed("proceventonfe-cancelamento.xml", 5, nfeChaveCancelada, nfeChaveProc)
	doc, err := e.repo.CompanyDocumentByChave(context.Background(), e.company.ID, sefaz.TpAmbProducao, nfeChaveProc)
	if err != nil {
		e.t.Fatal(err)
	}
	if doc.Situacao != nfe.SituacaoCancelada || doc.Completeness != nfe.CompletenessCompleta {
		e.t.Fatalf("seeded %s is (%s, %s), want a cancelled completa", nfeChaveProc, doc.Situacao, doc.Completeness)
	}
}

func sortedStrings(values ...string) []string {
	slices.Sort(values)
	return values
}

func TestNFeExportCSVAndXLSXMarkTheirOwnKind(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedFixtures()
	ctx := context.Background()
	outDir := t.TempDir()
	all := sortedStrings(nfeChaveProc, nfeChaveCancelada, nfeChaveDenegada)

	csvPath := filepath.Join(outDir, "nfe.csv")
	res, err := env.app.NFe.ExportCSV(ctx, NFeExportInput{CNPJ: nfeTestCNPJ, OutPath: csvPath})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 2 || res.SkippedResumos != 1 || res.Format != "csv" || res.OutPath != csvPath {
		t.Errorf("csv result = %+v, want 2 exported and 1 resumo skipped", res)
	}
	data, err := os.ReadFile(csvPath) // #nosec G304 -- test temp dir.
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(string(data), "\n"); lines != 3 {
		t.Errorf("csv has %d lines, want header + 2 rows:\n%s", lines, data)
	}
	if got := env.pendingChaves(nfe.ExportKindCSV); !slices.Equal(got, []string{nfeChaveCancelada}) {
		t.Errorf("csv pending = %v, want the resumo only", got)
	}
	if got := env.pendingChaves(nfe.ExportKindXLSX); !slices.Equal(got, all) {
		t.Errorf("xlsx pending after a csv export = %v, want all", got)
	}

	xlsxPath := filepath.Join(outDir, "nfe.xlsx")
	res, err = env.app.NFe.ExportXLSX(ctx, NFeExportInput{CNPJ: nfeTestCNPJ, IncludeResumos: true, Incremental: true, OutPath: xlsxPath})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 3 || res.SkippedResumos != 0 || res.Format != "xlsx" {
		t.Errorf("xlsx result = %+v, want 3 exported", res)
	}
	f, err := excelize.OpenFile(xlsxPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	rows, err := f.GetRows("NFe Recebidas")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Errorf("NFe Recebidas has %d rows, want header + 3", len(rows))
	}
	if got := env.pendingChaves(nfe.ExportKindXLSX); len(got) != 0 {
		t.Errorf("xlsx pending = %v, want none", got)
	}
	if got := env.pendingChaves(nfe.ExportKindXML); !slices.Equal(got, all) {
		t.Errorf("xml pending = %v, want all", got)
	}

	res, err = env.app.NFe.ExportCSV(ctx, NFeExportInput{CNPJ: nfeTestCNPJ, Incremental: true, OutPath: filepath.Join(outDir, "again.csv")})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 0 || res.OutPath != "" || res.SkippedResumos != 1 {
		t.Errorf("incremental csv = %+v, want nothing but the skipped resumo", res)
	}
}

func TestNFeExportDANFEZipSkipsResumos(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedCancelledCompleta()
	fake := env.useFakeDANFE()
	ctx := context.Background()
	outPath := filepath.Join(t.TempDir(), "danfes.zip")

	// IncludeResumos is ignored: a resumo has no DANFE.
	res, err := env.app.NFe.ExportDANFEZip(ctx, NFeExportInput{CNPJ: nfeTestCNPJ, IncludeResumos: true, OutPath: outPath})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 2 || res.SkippedResumos != 1 || res.Format != "danfe" || res.OutPath != outPath {
		t.Errorf("result = %+v, want 2 exported and 1 resumo skipped", res)
	}
	want := []string{
		"2026-09/destinatario/" + nfeChaveProc + ".pdf",
		"2026-09/destinatario/" + nfeChaveDenegada + ".pdf",
	}
	if got := sortedKeys(zipEntries(t, outPath)); !slices.Equal(got, want) {
		t.Errorf("zip entries = %v, want %v", got, want)
	}
	if len(fake.calls) != 2 {
		t.Fatalf("renders = %d, want 2", len(fake.calls))
	}
	if !fake.cancelledFor(t, nfeChaveProc) {
		t.Error("the cancelled NF-e was rendered without Cancelada")
	}
	if fake.cancelledFor(t, nfeChaveDenegada) {
		t.Error("the denegada NF-e was rendered with Cancelada")
	}
	if got := env.pendingChaves(nfe.ExportKindDANFE); !slices.Equal(got, []string{nfeChaveCancelada}) {
		t.Errorf("danfe pending = %v, want the resumo only", got)
	}
}

func TestNFeExportDANFE(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedCancelledCompleta()
	fake := env.useFakeDANFE()
	ctx := context.Background()
	outDir := t.TempDir()

	resumoPath := filepath.Join(outDir, "resumo.pdf")
	err := env.app.NFe.ExportDANFE(ctx, NFeExportDANFEInput{CNPJ: nfeTestCNPJ, ChaveAcesso: nfeChaveCancelada, OutPath: resumoPath})
	wantErr := "a NF-e " + nfeChaveCancelada + " é um resumo: o DANFE precisa do XML completo"
	if err == nil || err.Error() != wantErr {
		t.Errorf("DANFE of a resumo = %v, want %q", err, wantErr)
	}
	if _, err := os.Stat(resumoPath); !os.IsNotExist(err) {
		t.Errorf("DANFE of a resumo wrote a file: %v", err)
	}

	outPath := filepath.Join(outDir, "danfe.pdf")
	if err := env.app.NFe.ExportDANFE(ctx, NFeExportDANFEInput{CNPJ: nfeTestCNPJ, ChaveAcesso: nfeChaveProc, OutPath: outPath}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(outPath) // #nosec G304 -- test temp dir.
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "%PDF-fake" {
		t.Errorf("DANFE = %q, want the rendered PDF", data)
	}
	if len(fake.calls) != 1 || !fake.calls[0].opts.Cancelada {
		t.Errorf("renders = %+v, want one with Cancelada", fake.calls)
	}
	if string(fake.calls[0].xml) != readNFeFixture(t, "procnfe.xml") {
		t.Error("the renderer did not get the stored procNFe")
	}
	if got := env.pendingChaves(nfe.ExportKindDANFE); !slices.Equal(got, sortedStrings(nfeChaveCancelada, nfeChaveDenegada)) {
		t.Errorf("danfe pending = %v, want the exported chave marked", got)
	}
	if got := env.pendingChaves(nfe.ExportKindXML); len(got) != 3 {
		t.Errorf("xml pending = %v, want all three", got)
	}
}

func TestNFeExportDANFEFailingRendererLeavesNothing(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedFixtures()
	fake := env.useFakeDANFE()
	fake.err = errors.New("layout quebrado")
	ctx := context.Background()
	outDir := t.TempDir()
	all := sortedStrings(nfeChaveProc, nfeChaveCancelada, nfeChaveDenegada)

	zipPath := filepath.Join(outDir, "danfes.zip")
	if _, err := env.app.NFe.ExportDANFEZip(ctx, NFeExportInput{CNPJ: nfeTestCNPJ, OutPath: zipPath}); err == nil || !strings.Contains(err.Error(), "layout quebrado") {
		t.Errorf("ExportDANFEZip = %v, want the render error", err)
	}
	pdfPath := filepath.Join(outDir, "danfe.pdf")
	if err := env.app.NFe.ExportDANFE(ctx, NFeExportDANFEInput{CNPJ: nfeTestCNPJ, ChaveAcesso: nfeChaveProc, OutPath: pdfPath}); err == nil || !strings.Contains(err.Error(), "layout quebrado") {
		t.Errorf("ExportDANFE = %v, want the render error", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("failed exports left files: %v", entries)
	}
	if got := env.pendingChaves(nfe.ExportKindDANFE); !slices.Equal(got, all) {
		t.Errorf("danfe pending = %v, want all after failures", got)
	}
}

func TestNFeExportDANFEWithoutRenderer(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedFixtures()
	ctx := context.Background()
	outDir := t.TempDir()

	if _, err := env.app.NFe.ExportDANFEZip(ctx, NFeExportInput{CNPJ: nfeTestCNPJ, OutPath: filepath.Join(outDir, "danfes.zip")}); err == nil || err.Error() != "DANFE não configurado" {
		t.Errorf("ExportDANFEZip = %v, want DANFE não configurado", err)
	}
	if err := env.app.NFe.ExportDANFE(ctx, NFeExportDANFEInput{CNPJ: nfeTestCNPJ, ChaveAcesso: nfeChaveProc, OutPath: filepath.Join(outDir, "danfe.pdf")}); err == nil || err.Error() != "DANFE não configurado" {
		t.Errorf("ExportDANFE = %v, want DANFE não configurado", err)
	}
}

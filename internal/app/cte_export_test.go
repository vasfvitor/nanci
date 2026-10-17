package app

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/dacte"
)

func TestCTeExportXMLZipLayoutAndIncrementalMarks(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedCTeFixtures()
	ctx := context.Background()
	outDir := t.TempDir()

	first := filepath.Join(outDir, "cte.zip")
	res, err := env.app.CTe.ExportXMLZip(ctx, CTeExportInput{CNPJ: nfeTestCNPJ, Incremental: true, OutPath: first})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 5 || res.OutPath != first || res.Format != cte.ExportKindXML || !res.Incremental {
		t.Errorf("result = %+v, want the 5 produção CT-e", res)
	}
	entries := zipEntries(t, first)
	procEntry := "2026-09/tomador/" + cteChaveProc + "-procCTe.xml"
	want := []string{
		"2026-09/autorizado/" + cteChaveToma4 + "-procCTe.xml",
		"2026-09/autorizado/eventos/" + cteChaveToma4 + "-110180-1.xml",
		"2026-09/tomador/" + cteChaveProc + "-procCTe.xml",
		"2026-09/tomador/" + cteChaveOS + "-procCTeOS.xml",
		"2026-09/tomador/" + cteChaveGTVe + "-procGTVe.xml",
		"2026-09/tomador/" + cteChaveSimp + "-procCTeSimp.xml",
		"2026-09/tomador/eventos/" + cteChaveProc + "-110111-1.xml",
	}
	slices.Sort(want)
	if got := sortedKeys(entries); !slices.Equal(got, want) {
		t.Fatalf("zip entries =\n%v\nwant\n%v", got, want)
	}
	if entries[procEntry] != readCTeFixture(t, "procte.xml") {
		t.Error("procCTe entry is not the stored XML")
	}

	second := filepath.Join(outDir, "again.zip")
	res, err = env.app.CTe.ExportXMLZip(ctx, CTeExportInput{CNPJ: nfeTestCNPJ, Incremental: true, OutPath: second})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 0 || res.OutPath != "" {
		t.Errorf("second incremental = %+v, want nothing exported", res)
	}
	if _, err := os.Stat(second); !os.IsNotExist(err) {
		t.Errorf("an empty export wrote %s", second)
	}

	full := filepath.Join(outDir, "full.zip")
	res, err = env.app.CTe.ExportXMLZip(ctx, CTeExportInput{CNPJ: nfeTestCNPJ, Role: "destinatario", OutPath: full})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 2 {
		t.Errorf("non-incremental export as destinatário = %+v, want procte and the GTV-e again", res)
	}

	single := filepath.Join(outDir, "single.xml")
	if err := env.app.CTe.ExportXML(ctx, CTeExportXMLInput{CNPJ: nfeTestCNPJ, ChaveAcesso: cteChaveGTVe, OutPath: single}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(single) // #nosec G304 -- test temp dir.
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != readCTeFixture(t, "procgtve.xml") {
		t.Error("single export is not the stored XML")
	}
	if err := env.app.CTe.ExportXML(ctx, CTeExportXMLInput{CNPJ: nfeTestCNPJ, ChaveAcesso: cteChaveV200, OutPath: single}); err == nil {
		t.Error("ExportXML of a homologação CT-e from produção succeeded, want not found")
	}
}

// An event stored in the same second as the export mark is still exported
// by the next incremental export, and only once.
func TestCTeExportXMLZipIncrementalSeesAnEventOfTheSameSecond(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedCTe(env.company.ID, env.company.CNPJ, "procte.xml", 1)
	ctx := context.Background()
	outDir := t.TempDir()

	first := filepath.Join(outDir, "first.zip")
	res, err := env.app.CTe.ExportXMLZip(ctx, CTeExportInput{CNPJ: nfeTestCNPJ, Incremental: true, OutPath: first})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 1 {
		t.Fatalf("first incremental = %+v, want the procCTe", res)
	}

	env.seedCTe(env.company.ID, env.company.CNPJ, "proceventocte-cancelamento.xml", 2)
	if _, err := env.db.ExecContext(ctx, `UPDATE cte_events SET created_at = (SELECT exported_at FROM company_cte_export_marks)`); err != nil {
		t.Fatal(err)
	}

	second := filepath.Join(outDir, "second.zip")
	res, err = env.app.CTe.ExportXMLZip(ctx, CTeExportInput{CNPJ: nfeTestCNPJ, Incremental: true, OutPath: second})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 1 {
		t.Fatalf("second incremental = %+v, want the procCTe again", res)
	}
	want := []string{
		"2026-09/tomador/" + cteChaveProc + "-procCTe.xml",
		"2026-09/tomador/eventos/" + cteChaveProc + "-110111-1.xml",
	}
	if got := sortedKeys(zipEntries(t, second)); !slices.Equal(got, want) {
		t.Fatalf("zip entries = %v, want %v", got, want)
	}

	third := filepath.Join(outDir, "third.zip")
	res, err = env.app.CTe.ExportXMLZip(ctx, CTeExportInput{CNPJ: nfeTestCNPJ, Incremental: true, OutPath: third})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 0 {
		t.Errorf("third incremental = %+v, want nothing exported", res)
	}
}

// fakeDACTE renders a PDF stand-in that says whether the watermark was
// asked for, or fails with err.
type fakeDACTE struct {
	err error
}

func (f *fakeDACTE) Render(_ []byte, opts dacte.Options) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return fmt.Appendf(nil, "%%PDF cancelada=%t", opts.Cancelada), nil
}

// ctePendingChaves lists, sorted, the produção CT-e not yet exported under
// kind.
func ctePendingChaves(t *testing.T, env *nfeTestEnv, kind string) []string {
	t.Helper()
	filter, err := cteFilter(env.company, ListCTeInput{})
	if err != nil {
		t.Fatal(err)
	}
	docs, err := cteRepo(env).ListPendingExport(context.Background(), env.company.ID, filter, kind)
	if err != nil {
		t.Fatal(err)
	}
	chaves := cteChavesOf(docs)
	slices.Sort(chaves)
	return chaves
}

// Only the CT-e modelo 57 proper get a DACTE: the CT-e OS, GTV-e and CT-e
// Simplificado are counted, left out and stay pending, so the next
// incremental export reports them again.
func TestCTeExportDACTEZipSkipsKindsWithoutDACTE(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedCTeFixtures()
	env.app.CTe.DACTERenderer = &fakeDACTE{}
	ctx := context.Background()
	outDir := t.TempDir()

	first := filepath.Join(outDir, "dactes.zip")
	res, err := env.app.CTe.ExportDACTEZip(ctx, CTeExportInput{CNPJ: nfeTestCNPJ, Incremental: true, OutPath: first})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 2 || res.SkippedSemDACTE != 3 || res.OutPath != first || res.Format != cte.ExportKindDACTE {
		t.Errorf("result = %+v, want 2 DACTE and 3 skipped", res)
	}
	want := map[string]string{
		"2026-09/tomador/" + cteChaveProc + ".pdf":     "%PDF cancelada=true",
		"2026-09/autorizado/" + cteChaveToma4 + ".pdf": "%PDF cancelada=false",
	}
	if got := zipEntries(t, first); !maps.Equal(got, want) {
		t.Errorf("zip entries = %v, want %v", got, want)
	}

	skipped := []string{cteChaveOS, cteChaveGTVe, cteChaveSimp}
	slices.Sort(skipped)
	if got := ctePendingChaves(t, env, cte.ExportKindDACTE); !slices.Equal(got, skipped) {
		t.Errorf("pending DACTE = %v, want the skipped kinds %v", got, skipped)
	}

	second := filepath.Join(outDir, "again.zip")
	res, err = env.app.CTe.ExportDACTEZip(ctx, CTeExportInput{CNPJ: nfeTestCNPJ, Incremental: true, OutPath: second})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 0 || res.SkippedSemDACTE != 3 || res.OutPath != "" {
		t.Errorf("second incremental = %+v, want nothing exported and 3 skipped again", res)
	}
	if _, err := os.Stat(second); !os.IsNotExist(err) {
		t.Errorf("an export with only skipped documents wrote %s", second)
	}
}

func TestCTeExportDACTESingle(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedCTeFixtures()
	env.app.CTe.DACTERenderer = &fakeDACTE{}
	ctx := context.Background()
	outDir := t.TempDir()

	out := filepath.Join(outDir, "proc.pdf")
	if err := env.app.CTe.ExportDACTE(ctx, CTeExportDACTEInput{CNPJ: nfeTestCNPJ, ChaveAcesso: cteChaveProc, OutPath: out}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out) // #nosec G304 -- test temp dir.
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "%PDF cancelada=true" {
		t.Errorf("DACTE of the cancelled CT-e = %q, want the watermark", data)
	}
	if slices.Contains(ctePendingChaves(t, env, cte.ExportKindDACTE), cteChaveProc) {
		t.Error("single DACTE left the CT-e pending")
	}

	osOut := filepath.Join(outDir, "os.pdf")
	err = env.app.CTe.ExportDACTE(ctx, CTeExportDACTEInput{CNPJ: nfeTestCNPJ, ChaveAcesso: cteChaveOS, OutPath: osOut})
	if err == nil || !strings.Contains(err.Error(), "DACTE disponível só para o CT-e modelo 57") {
		t.Errorf("DACTE of a CT-e OS: err = %v, want the modelo 57 error", err)
	}
	if _, err := os.Stat(osOut); !os.IsNotExist(err) {
		t.Errorf("DACTE of a CT-e OS wrote %s", osOut)
	}
}

func TestCTeExportDACTEFailingRendererWritesAndMarksNothing(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedCTeFixtures()
	env.app.CTe.DACTERenderer = &fakeDACTE{err: errors.New("layout quebrado")}
	ctx := context.Background()
	outDir := t.TempDir()

	out := filepath.Join(outDir, "dactes.zip")
	if _, err := env.app.CTe.ExportDACTEZip(ctx, CTeExportInput{CNPJ: nfeTestCNPJ, Incremental: true, OutPath: out}); err == nil {
		t.Fatal("ExportDACTEZip with a failing renderer succeeded")
	}
	single := filepath.Join(outDir, "proc.pdf")
	if err := env.app.CTe.ExportDACTE(ctx, CTeExportDACTEInput{CNPJ: nfeTestCNPJ, ChaveAcesso: cteChaveProc, OutPath: single}); err == nil {
		t.Fatal("ExportDACTE with a failing renderer succeeded")
	}
	if entries, err := os.ReadDir(outDir); err != nil || len(entries) != 0 {
		t.Errorf("failed exports left files: %v (err %v)", entries, err)
	}
	if got := ctePendingChaves(t, env, cte.ExportKindDACTE); len(got) != 5 {
		t.Errorf("pending DACTE = %v, want all 5 still pending", got)
	}
}

func TestCTeExportDACTEWithoutRenderer(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedCTeFixtures()
	ctx := context.Background()
	out := filepath.Join(t.TempDir(), "dactes.zip")

	if _, err := env.app.CTe.ExportDACTEZip(ctx, CTeExportInput{CNPJ: nfeTestCNPJ, OutPath: out}); err == nil || err.Error() != "DACTE não configurado" {
		t.Errorf("ExportDACTEZip without renderer: err = %v", err)
	}
	if err := env.app.CTe.ExportDACTE(ctx, CTeExportDACTEInput{CNPJ: nfeTestCNPJ, ChaveAcesso: cteChaveProc, OutPath: out}); err == nil || err.Error() != "DACTE não configurado" {
		t.Errorf("ExportDACTE without renderer: err = %v", err)
	}
}

// Each format keeps its own marks: exporting the CSV does not mark the XLSX
// or the XML as exported.
func TestCTeExportSpreadsheetsMarkPerKind(t *testing.T) {
	env := newNFeTestEnv(t)
	env.seedCTeFixtures()
	ctx := context.Background()
	outDir := t.TempDir()

	csvOut := filepath.Join(outDir, "cte.csv")
	res, err := env.app.CTe.ExportCSV(ctx, CTeExportInput{CNPJ: nfeTestCNPJ, Incremental: true, OutPath: csvOut})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 5 || res.Format != cte.ExportKindCSV || res.SkippedSemDACTE != 0 {
		t.Errorf("csv = %+v, want the 5 produção CT-e", res)
	}
	data, err := os.ReadFile(csvOut) // #nosec G304 -- test temp dir.
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(string(data), "\n"); lines != 6 {
		t.Errorf("csv has %d lines, want a header and 5 rows", lines)
	}

	res, err = env.app.CTe.ExportCSV(ctx, CTeExportInput{CNPJ: nfeTestCNPJ, Incremental: true, OutPath: csvOut})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 0 {
		t.Errorf("second csv = %+v, want nothing pending", res)
	}

	xlsxOut := filepath.Join(outDir, "cte.xlsx")
	res, err = env.app.CTe.ExportXLSX(ctx, CTeExportInput{CNPJ: nfeTestCNPJ, Incremental: true, OutPath: xlsxOut})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExportedCount != 5 || res.Format != cte.ExportKindXLSX {
		t.Errorf("xlsx after csv = %+v, want the 5 CT-e still pending for xlsx", res)
	}
	if _, err := os.Stat(xlsxOut); err != nil {
		t.Errorf("xlsx not written: %v", err)
	}
	if got := ctePendingChaves(t, env, cte.ExportKindXML); len(got) != 5 {
		t.Errorf("pending XML = %v, want all 5", got)
	}
}

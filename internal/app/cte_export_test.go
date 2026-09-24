package app

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/vasfvitor/nanci/internal/cte"
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

package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/sync"
)

// Fixtures of internal/nfse/testdata. seedNFSeFixture puts the test company
// in the place of their synthetic party 12345678000199.
const (
	nfseFixtureCNPJ   = "12345678000199"
	nfseChavePrestada = "11223344556677889900112233445566778899001122441111" // simple-prestada.xml
	nfseChaveTomada   = "99887766554433221100998877665544332211009988779999" // simple-tomada.xml
)

// seedNFSeFixture applies internal/nfse/testdata/<fixture>.xml to the company
// as a sync would, with the company as the fixture's party 12345678000199,
// and returns the stored XML. It also lets the company see documents of any
// date, since the fixtures are older than the company.
func (e *nfeTestRoot) seedNFSeFixture(fixture string) []byte {
	e.t.Helper()
	ctx := context.Background()
	raw, err := os.ReadFile(filepath.Join("..", "nfse", "testdata", fixture+".xml")) // #nosec G304 -- fixed testdata path.
	if err != nil {
		e.t.Fatal(err)
	}
	data := []byte(strings.ReplaceAll(string(raw), nfseFixtureCNPJ, e.company.CNPJ))

	doc, warnings, err := nfse.ParseDocumentXML(data)
	if err != nil {
		e.t.Fatalf("parse %s: %v", fixture, err)
	}
	sum := sha256.Sum256(data)
	doc.ID = nfse.DocumentID("doc-" + string(doc.ChaveAcesso))
	doc.RawHash = hex.EncodeToString(sum[:])
	doc.ParseWarnings = warnings
	if err := e.xml.Store(doc.RawHash, data); err != nil {
		e.t.Fatal(err)
	}

	if _, err := sync.NewStore(e.db).ApplyDocument(ctx, nfse.ApplyDocumentParams{
		Document:      doc,
		Participation: nfse.ClassifyCompanyParticipation(&doc, e.company.CNPJ),
		CompanyID:     e.company.ID,
		NSU:           1,
	}); err != nil {
		e.t.Fatalf("apply %s: %v", fixture, err)
	}
	if _, err := e.db.ExecContext(ctx, `UPDATE companies SET sync_start_policy = 'all', sync_start_date = NULL WHERE id = ?`, string(e.company.ID)); err != nil {
		e.t.Fatal(err)
	}
	return data
}

func TestExportXLSX_RepeatedChave(t *testing.T) {
	env := newNFeTestRoot(t)
	env.seedNFSeFixture("simple-prestada")
	env.seedNFSeFixture("simple-tomada")
	env.seedNFSeFixture("com-numero-descricao")
	out := filepath.Join(t.TempDir(), "notas.xlsx")

	if err := env.run("export", "xlsx", "-c", nfeTestCNPJ, "--chave", nfseChavePrestada, "--chave", nfseChaveTomada, "-o", out); err != nil {
		t.Fatalf("export xlsx: %v", err)
	}
	if !strings.Contains(env.out.String(), "Documentos exportados: 2") {
		t.Errorf("output = %q, want 2 documents", env.out.String())
	}

	f, err := excelize.OpenFile(out)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	// The workbook has one sheet per papel; count the document rows of all.
	var chaves []string
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range rows {
			for _, cell := range row {
				if len(cell) == 50 {
					chaves = append(chaves, cell)
				}
			}
		}
	}
	slices.Sort(chaves)
	if want := []string{nfseChavePrestada, nfseChaveTomada}; !slices.Equal(chaves, want) {
		t.Errorf("chaves in the workbook = %v, want %v", chaves, want)
	}
}

func TestExport_InvalidChaveSkipsAppFactory(t *testing.T) {
	for _, args := range [][]string{
		{"export", "csv", "-c", nfeTestCNPJ, "--chave", "123"},
		{"export", "danfse-zip", "-c", nfeTestCNPJ, "--chave", nfseChavePrestada, "--chave", "123"},
		{"export", "xml", "-c", nfeTestCNPJ, "--chave", "123"},
	} {
		calls := 0
		v, tr := false, false
		root := NewRootCommand(CommandEnv{
			Stdin:  os.Stdin,
			Stderr: os.Stderr,
			Stdout: &bytes.Buffer{},
			AppFactory: func(context.Context) (*app.App, func(), error) {
				calls++
				return nil, nil, errors.New("AppFactory must not be called")
			},
			Verbose: &v,
			Trace:   &tr,
		})
		root.SetErr(&bytes.Buffer{})
		root.SetArgs(args)

		err := root.ExecuteContext(context.Background())
		if err == nil || !strings.Contains(err.Error(), `chave de acesso inválida "123"`) {
			t.Errorf("%v: err = %v, want the invalid-chave error", args[1], err)
		}
		if calls != 0 {
			t.Errorf("%v: AppFactory called %d times, want 0", args[1], calls)
		}
	}
}

func TestExportXML_WritesStoredXML(t *testing.T) {
	env := newNFeTestRoot(t)
	want := env.seedNFSeFixture("simple-prestada")
	out := filepath.Join(t.TempDir(), "nota.xml")

	if err := env.run("export", "xml", "-c", nfeTestCNPJ, "--chave", nfseChavePrestada, "-o", out); err != nil {
		t.Fatalf("export xml: %v", err)
	}
	if got := env.out.String(); got != "XML exportado com sucesso: "+out+"\n" {
		t.Errorf("output = %q", got)
	}
	got, err := os.ReadFile(out) // #nosec G304 -- test temp file.
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Error("exported XML differs from the stored one")
	}
}

func TestExportCSV_IncrementalTwice(t *testing.T) {
	env := newNFeTestRoot(t)
	env.seedNFSeFixture("simple-prestada")
	env.seedNFSeFixture("simple-tomada")
	out := filepath.Join(t.TempDir(), "notas.csv")

	if err := env.run("export", "csv", "-c", nfeTestCNPJ, "--incremental", "-o", out); err != nil {
		t.Fatalf("first export: %v", err)
	}
	if !strings.Contains(env.out.String(), "Documentos exportados: 2") {
		t.Errorf("first output = %q, want 2 documents", env.out.String())
	}

	if err := env.run("export", "csv", "-c", nfeTestCNPJ, "--incremental", "-o", out); err != nil {
		t.Fatalf("second export: %v", err)
	}
	if !strings.Contains(env.out.String(), "Nenhum documento pendente para exportação incremental.") {
		t.Errorf("second output = %q, want nothing pending", env.out.String())
	}
}

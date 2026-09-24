package cli

import (
	"archive/zip"
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
	"time"

	"github.com/vasfvitor/nanci/internal/app"
	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store"
	"github.com/vasfvitor/nanci/internal/sync"
)

// Fixtures of internal/cte/testdata. The test company (nfeTestCNPJ) is the
// tomador of all three.
const (
	cteChaveProc   = "35260912345678000195570010000001011123456784" // procte.xml (tomador and destinatário), proceventocte-cancelamento.xml
	cteChaveOS     = "35260912345678000195670010000001041456789014" // procteos.xml (tomador)
	cteChaveToma03 = "35260912345678000195570010000001031345678907" // procte-v200-toma03.xml (tomador and remetente, homologação)
	cteNFeChave    = "35260911222333000181550010000012341123456787" // an NF-e procte.xml transported
)

type cteTestRoot struct {
	*nfeTestRoot
	cteRepo *store.CTeRepository
}

// newCTeTestRoot builds a root over a real database holding one company, the
// tomador of the CT-e fixtures.
func newCTeTestRoot(t *testing.T) *cteTestRoot {
	t.Helper()
	env := newNFeTestRoot(t)
	return &cteTestRoot{nfeTestRoot: env, cteRepo: store.NewCTeRepository(env.db)}
}

// seedCTe stores the fixture as if the CT-e distribution had delivered it at
// nsu.
func (e *cteTestRoot) seedCTe(fixture string, nsu int64) {
	e.t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "cte", "testdata", fixture)) // #nosec G304 -- fixed testdata path.
	if err != nil {
		e.t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	if err := e.xml.Store(hash, data); err != nil {
		e.t.Fatal(err)
	}

	ctx := context.Background()
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		e.t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()

	if strings.HasPrefix(fixture, "proceventocte") {
		ev, err := cte.ParseProcEventoCTe(data)
		if err != nil {
			e.t.Fatalf("parse %s: %v", fixture, err)
		}
		ev.RawHash = hash
		if _, err := e.cteRepo.ApplyEventTx(ctx, tx, store.ApplyCTeEventParams{Event: ev}); err != nil {
			e.t.Fatal(err)
		}
	} else {
		doc, err := cte.ParseProcCTe(data)
		if err != nil {
			e.t.Fatalf("parse %s: %v", fixture, err)
		}
		doc.RawHash = hash
		params := store.ApplyCTeDocumentParams{Document: doc, CompanyID: e.company.ID, CompanyCNPJ: e.company.CNPJ, NSU: nsu}
		if _, err := e.cteRepo.ApplyDocumentTx(ctx, tx, params); err != nil {
			e.t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		e.t.Fatal(err)
	}
}

func TestCTeList_RejectsInvalidFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"modelo 58", []string{"--modelo", "58"}, "modelo inválido"},
		{"situacao", []string{"--situacao", "anulada"}, "situação inválida"},
		{"papel", []string{"--papel", "transportador"}, "papel inválido"},
		{"nfe key", []string{"--nfe", "3526091122233300018155001000001234112345678"}, "chave de NF-e inválida"},
		{"chave", []string{"--chave", cteChaveOS[:43] + "5"}, "chave de acesso inválida"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newCTeTestRoot(t) // flag values stick to a command tree
			err := env.run(append([]string{"cte", "list", "-c", nfeTestCNPJ}, tc.args...)...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Execute = %v, want an error containing %q", err, tc.want)
			}
		})
	}
}

// seedListFixtures stores a cancelled CT-e and a CT-e OS, both taken by the
// company.
func (e *cteTestRoot) seedListFixtures() {
	e.t.Helper()
	e.seedCTe("procte.xml", 1)
	e.seedCTe("proceventocte-cancelamento.xml", 2)
	e.seedCTe("procteos.xml", 3)
}

func TestCTeList_PrintsColumns(t *testing.T) {
	env := newCTeTestRoot(t)
	env.seedListFixtures()

	if err := env.run("cte", "list", "-c", nfeTestCNPJ); err != nil {
		t.Fatalf("list: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(env.out.String()), "\n")
	header := strings.Join(strings.Fields(lines[0]), " ")
	if want := "CHAVE DE ACESSO MODELO NÚMERO/SÉRIE EMISSÃO EMITENTE NOME EMITENTE TOMADOR PAPEL VALOR (R$) SITUAÇÃO"; header != want {
		t.Errorf("header = %q, want %q", header, want)
	}

	// Rows come newest first. The emitente name has spaces, so the columns
	// after it are read from the end of the line.
	wantRows := [][]string{
		{cteChaveOS, "67", "104/1", "2026-09-05", "12.345.678/0001-95", "70.860.312/0001-50", "tomador", "2.000,00", "autorizada"},
		{cteChaveProc, "57", "101/1", "2026-09-02", "12.345.678/0001-95", "70.860.312/0001-50", "tomador,destinatario", "1.500,00", "cancelada"},
	}
	rows := lines[2 : len(lines)-2]
	if len(rows) != len(wantRows) {
		t.Fatalf("rows = %d, want %d:\n%s", len(rows), len(wantRows), env.out.String())
	}
	for i, want := range wantRows {
		f := strings.Fields(rows[i])
		n := len(f)
		got := []string{f[0], f[1], f[2], f[3], f[4], f[n-4], f[n-3], f[n-2], f[n-1]}
		if !slices.Equal(got, want) {
			t.Errorf("row %d = %q\ngot  %v\nwant %v", i, rows[i], got, want)
		}
		if !strings.Contains(rows[i], "TRANSPORTADORA FICTICIA LTDA") {
			t.Errorf("row %d lacks the emitente name: %q", i, rows[i])
		}
	}
	if last := lines[len(lines)-1]; last != "Total de 2 CT-e listado(s)." {
		t.Errorf("footer = %q", last)
	}
}

func TestCTeList_Filters(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"--modelo", "67"}, cteChaveOS},
		{[]string{"--nfe", cteNFeChave}, cteChaveProc},
		{[]string{"--situacao", "cancelada"}, cteChaveProc},
		{[]string{"-p", "destinatario"}, cteChaveProc},
		{[]string{"--chave", cteChaveOS}, cteChaveOS},
		{[]string{"--chave", " " + cteChaveOS + " "}, cteChaveOS},
	}
	for _, tc := range tests {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			env := newCTeTestRoot(t) // flag values stick to a command tree
			env.seedListFixtures()
			if err := env.run(append([]string{"cte", "list", "-c", nfeTestCNPJ}, tc.args...)...); err != nil {
				t.Fatalf("list: %v", err)
			}
			if got := env.out.String(); !strings.Contains(got, tc.want) || !strings.Contains(got, "Total de 1 CT-e listado(s).") {
				t.Errorf("want only %s:\n%s", tc.want, got)
			}
		})
	}
}

func TestCTeList_PapelMatchesAnyRole(t *testing.T) {
	env := newCTeTestRoot(t)
	env.seedCTe("procte-v200-toma03.xml", 1)
	// The fixture is from homologação; the company lists only its environment.
	if err := env.run("company", "update", "-c", nfeTestCNPJ, "--env", "producao_restrita"); err != nil {
		t.Fatalf("company update --env producao_restrita: %v", err)
	}

	if err := env.run("cte", "list", "-c", nfeTestCNPJ, "--papel", "remetente"); err != nil {
		t.Fatalf("list --papel remetente: %v", err)
	}
	got := env.out.String()
	if !strings.Contains(got, cteChaveToma03) || !strings.Contains(got, "tomador,remetente") {
		t.Errorf("list --papel remetente lacks the tomador CT-e:\n%s", got)
	}

	if err := env.run("cte", "list", "-c", nfeTestCNPJ, "--papel", "destinatario"); err != nil {
		t.Fatalf("list --papel destinatario: %v", err)
	}
	if got := env.out.String(); got != "Nenhum CT-e encontrado.\n" {
		t.Errorf("list --papel destinatario = %q, want no CT-e", got)
	}
}

func TestCTeStatus_PrintsTotals(t *testing.T) {
	env := newCTeTestRoot(t)
	env.seedCTe("procte.xml", 1)
	env.seedCTe("procteos.xml", 2)

	if err := env.run("cte", "status", "-c", nfeTestCNPJ); err != nil {
		t.Fatalf("status: %v", err)
	}
	got := env.out.String()
	for _, want := range []string{"Ambiente: Produção (tpAmb 1) | UF: SP", "Carga inicial: pendente", "Como tomadora: 2", "Como destinatária: 0", "Como remetente: 0", "Outros papéis: 0"} {
		if !strings.Contains(got, want) {
			t.Errorf("status output lacks %q:\n%s", want, got)
		}
	}
}

func TestCTePull_BlockedPrintsNextAllowedAt(t *testing.T) {
	env := newCTeTestRoot(t)
	until := time.Now().Add(30 * time.Minute).UTC().Truncate(time.Second)
	if err := sync.NewStore(env.db).SetBlockedUntil(context.Background(), env.company.ID, nfse.SyncSourceCTe, until, nfse.SyncStopReasonCaughtUp); err != nil {
		t.Fatal(err)
	}

	err := env.run("cte", "pull", "-c", nfeTestCNPJ)
	if !errors.Is(err, app.ErrSourceBlocked) {
		t.Fatalf("Execute = %v, want ErrSourceBlocked", err)
	}
	want := "Próxima consulta permitida após: " + until.Local().Format("2006-01-02 15:04") + "\n"
	if got := env.out.String(); got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if env.passwords.requests != 0 {
		t.Errorf("password requests = %d, want 0", env.passwords.requests)
	}
}

func TestCTeReset_DryRunThenConfirm(t *testing.T) {
	env := newCTeTestRoot(t)
	env.seedCTe("procte.xml", 1)
	env.seedCTe("proceventocte-cancelamento.xml", 2)

	if err := env.run("cte", "reset", "-c", nfeTestCNPJ); err != nil {
		t.Fatalf("reset dry-run: %v", err)
	}
	got := env.out.String()
	for _, want := range []string{"CT-e da empresa (seriam removidos): 1", "Eventos (seriam removidos): 1", "Nada foi alterado. Use --confirmar"} {
		if !strings.Contains(got, want) {
			t.Errorf("dry-run output lacks %q:\n%s", want, got)
		}
	}
	if err := env.run("cte", "list", "-c", nfeTestCNPJ); err != nil || !strings.Contains(env.out.String(), cteChaveProc) {
		t.Fatalf("the dry-run removed the CT-e: %v\n%s", err, env.out.String())
	}

	if err := env.run("cte", "reset", "-c", nfeTestCNPJ, "--confirmar"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if got := env.out.String(); !strings.Contains(got, "CT-e da empresa (removidos): 1") || !strings.Contains(got, "Eventos (removidos): 1") {
		t.Errorf("reset output:\n%s", got)
	}
	docs, err := env.cteRepo.ListCompanyDocuments(context.Background(), env.company.ID, cte.DocumentFilter{})
	if err != nil || len(docs) != 0 {
		t.Errorf("documents after the reset = %d, %v; want none", len(docs), err)
	}
}

func TestCTeExportZip_Layout(t *testing.T) {
	env := newCTeTestRoot(t)
	env.seedCTe("procte.xml", 1)
	env.seedCTe("proceventocte-cancelamento.xml", 2)
	env.seedCTe("procteos.xml", 3)
	outPath := filepath.Join(t.TempDir(), "cte.zip")

	if err := env.run("cte", "export", "zip", "-c", nfeTestCNPJ, "--out", outPath); err != nil {
		t.Fatalf("export zip: %v", err)
	}
	if got := env.out.String(); !strings.Contains(got, "Arquivo xml gerado com sucesso: "+outPath) || !strings.Contains(got, "Documentos exportados: 2") {
		t.Errorf("export zip output:\n%s", got)
	}

	zr, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	slices.Sort(names)
	want := []string{
		"2026-09/tomador/" + cteChaveProc + "-procCTe.xml",
		"2026-09/tomador/" + cteChaveOS + "-procCTeOS.xml",
		"2026-09/tomador/eventos/" + cteChaveProc + "-110111-1.xml",
	}
	if !slices.Equal(names, want) {
		t.Errorf("zip entries = %v, want %v", names, want)
	}

	if err := env.run("cte", "export", "zip", "-c", nfeTestCNPJ, "--out", outPath, "--incremental"); err != nil {
		t.Fatalf("export zip --incremental: %v", err)
	}
	if got := env.out.String(); !strings.Contains(got, "Nenhum documento pendente para exportação incremental.") {
		t.Errorf("export zip --incremental output:\n%s", got)
	}
}

func TestCTeExportZip_ValidatesChave(t *testing.T) {
	env := newCTeTestRoot(t)
	env.seedListFixtures()
	outPath := filepath.Join(t.TempDir(), "cte.zip")

	err := env.run("cte", "export", "zip", "-c", nfeTestCNPJ, "--out", outPath, "--chave", "35260912345678000195")
	if !errors.Is(err, dfe.ErrInvalidAccessKey) {
		t.Fatalf("export zip with an invalid --chave = %v, want dfe.ErrInvalidAccessKey", err)
	}

	env = newCTeTestRoot(t) // flag values stick to a command tree
	env.seedListFixtures()
	if err := env.run("cte", "export", "zip", "-c", nfeTestCNPJ, "--out", outPath, "--chave", " "+cteChaveOS+" "); err != nil {
		t.Fatalf("export zip with a spaced --chave: %v", err)
	}
	if got := env.out.String(); !strings.Contains(got, "Documentos exportados: 1") {
		t.Errorf("export zip output:\n%s", got)
	}
}

func TestCTeExportXML_WritesStoredXML(t *testing.T) {
	env := newCTeTestRoot(t)
	env.seedCTe("procteos.xml", 1)
	outPath := filepath.Join(t.TempDir(), "os.xml")

	if err := env.run("cte", "export", "xml", "-c", nfeTestCNPJ, "--chave", cteChaveOS, "--out", outPath); err != nil {
		t.Fatalf("export xml: %v", err)
	}
	got, err := os.ReadFile(outPath) // #nosec G304 -- test temp path.
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile(filepath.Join("..", "cte", "testdata", "procteos.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, fixture) {
		t.Error("exported XML differs from the stored fixture")
	}

	if err := env.run("cte", "export", "xml", "-c", nfeTestCNPJ, "--chave", "123"); err == nil || !strings.Contains(err.Error(), "chave de acesso inválida") {
		t.Errorf("export xml --chave 123 = %v, want an invalid chave error", err)
	}
}

package app

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/company"
	"github.com/vasfvitor/nanci/internal/credential"
	"github.com/vasfvitor/nanci/internal/dfe"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfse"
	"github.com/vasfvitor/nanci/internal/store/storetest"
	"github.com/vasfvitor/nanci/internal/sync"
)

// Fixtures of internal/nfse/testdata. seedNFSe puts the test company in the
// place of their synthetic party 12345678000199.
const (
	nfseFixtureCNPJ      = "12345678000199"
	nfseChavePrestada    = "11223344556677889900112233445566778899001122441111" // simple-prestada.xml, 2023-10
	nfseChaveTomada      = "99887766554433221100998877665544332211009988779999" // simple-tomada.xml, 2023-11
	nfseInvalidChaveText = "chave de acesso inválida"
)

// seedNFSe applies internal/nfse/testdata/<fixture>.xml to env's company as a
// sync would, with the company as the fixture's party 12345678000199, and
// returns the stored XML. It also lets the company see documents of any date,
// since the fixtures are older than the company.
func seedNFSe(t *testing.T, env *nfeTestEnv, fixture string) []byte {
	t.Helper()
	ctx := context.Background()
	raw, err := os.ReadFile(filepath.Join("..", "nfse", "testdata", fixture+".xml")) // #nosec G304 -- fixed testdata path.
	if err != nil {
		t.Fatal(err)
	}
	data := []byte(strings.ReplaceAll(string(raw), nfseFixtureCNPJ, env.company.CNPJ))

	doc, warnings, err := nfse.ParseDocumentXML(data)
	if err != nil {
		t.Fatalf("parse %s: %v", fixture, err)
	}
	sum := sha256.Sum256(data)
	doc.ID = nfse.DocumentID("doc-" + string(doc.ChaveAcesso))
	doc.RawHash = hex.EncodeToString(sum[:])
	doc.ParseWarnings = warnings
	if err := env.xml.Store(doc.RawHash, data); err != nil {
		t.Fatal(err)
	}

	if _, err := sync.NewStore(env.db).ApplyDocument(ctx, nfse.ApplyDocumentParams{
		Document:      doc,
		Participation: nfse.ClassifyCompanyParticipation(&doc, env.company.CNPJ),
		CompanyID:     env.company.ID,
		NSU:           1,
	}); err != nil {
		t.Fatalf("apply %s: %v", fixture, err)
	}
	if _, err := env.db.ExecContext(ctx, `UPDATE companies SET sync_start_policy = 'all', sync_start_date = NULL WHERE id = ?`, string(env.company.ID)); err != nil {
		t.Fatal(err)
	}
	return data
}

type fakeDANFSeRenderer struct {
	err error
}

func (r fakeDANFSeRenderer) Render(xml []byte) ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]byte("pdf:"), xml[:10]...), nil
}

func TestExport_InvalidChaveWritesNothing(t *testing.T) {
	env := newNFeTestEnv(t)
	seedNFSe(t, env, "simple-prestada")
	env.app.Exports.DANFSeRenderer = fakeDANFSeRenderer{}
	ctx := context.Background()
	dir := t.TempDir()

	bulk := map[string]func(context.Context, ExportInput) (ExportResult, error){
		"csv":    env.app.Exports.ExportCSV,
		"xlsx":   env.app.Exports.ExportXLSX,
		"zip":    env.app.Exports.ExportZIP,
		"danfse": env.app.Exports.ExportDANFSeZIP,
	}
	for name, export := range bulk {
		out := filepath.Join(dir, name+".out")
		_, err := export(ctx, ExportInput{CNPJ: env.company.CNPJ, OutPath: out, ChavesAcesso: []string{nfseChavePrestada, "123"}})
		if err == nil || !strings.Contains(err.Error(), nfseInvalidChaveText) {
			t.Errorf("%s: err = %v, want %q", name, err, nfseInvalidChaveText)
		}
		if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s: output exists after an invalid chave (stat err %v)", name, err)
		}
	}

	out := filepath.Join(dir, "single.xml")
	if err := env.app.Exports.ExportXML(ctx, ExportXMLInput{CNPJ: env.company.CNPJ, ChaveAcesso: "123", OutPath: out}); err == nil || !strings.Contains(err.Error(), nfseInvalidChaveText) {
		t.Errorf("ExportXML: err = %v, want %q", err, nfseInvalidChaveText)
	}
	if err := env.app.Exports.ExportDANFSe(ctx, ExportDANFSeInput{CNPJ: env.company.CNPJ, ChaveAcesso: "123", OutPath: out}); err == nil || !strings.Contains(err.Error(), nfseInvalidChaveText) {
		t.Errorf("ExportDANFSe: err = %v, want %q", err, nfseInvalidChaveText)
	}
	if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("single export wrote a file after an invalid chave (stat err %v)", err)
	}
}

func TestExport_PaddedChave(t *testing.T) {
	env := newNFeTestEnv(t)
	want := seedNFSe(t, env, "simple-prestada")
	seedNFSe(t, env, "simple-tomada")
	ctx := context.Background()
	padded := "  " + nfseChavePrestada + " "

	out := filepath.Join(t.TempDir(), "nota.xml")
	if err := env.app.Exports.ExportXML(ctx, ExportXMLInput{CNPJ: env.company.CNPJ, ChaveAcesso: padded, OutPath: out}); err != nil {
		t.Fatalf("ExportXML: %v", err)
	}
	got, err := os.ReadFile(out) // #nosec G304 -- test temp file.
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Error("exported XML differs from the stored one")
	}

	res, err := env.app.Exports.ExportCSV(ctx, ExportInput{CNPJ: env.company.CNPJ, OutPath: filepath.Join(t.TempDir(), "notas.csv"), ChavesAcesso: []string{padded}})
	if err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}
	if res.ExportedCount != 1 {
		t.Errorf("ExportedCount = %d, want 1", res.ExportedCount)
	}
}

func TestExportDANFSeZIP_RenderFailureLeavesNothing(t *testing.T) {
	env := newNFeTestEnv(t)
	seedNFSe(t, env, "simple-prestada")
	env.app.Exports.DANFSeRenderer = fakeDANFSeRenderer{err: errors.New("layout quebrado")}
	ctx := context.Background()
	dir := t.TempDir()
	out := filepath.Join(dir, "danfses.zip")

	_, err := env.app.Exports.ExportDANFSeZIP(ctx, ExportInput{CNPJ: env.company.CNPJ, OutPath: out, Incremental: true})
	if err == nil || !strings.Contains(err.Error(), "layout quebrado") {
		t.Fatalf("err = %v, want the renderer error", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("files left after a failed export: %v", entries)
	}
	pending, err := env.app.Exports.DocumentRepo.ListPendingExportDocuments(ctx, env.company.ID, nfse.DocumentFilter{}, nfse.ExportKindDANFSe)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Errorf("pending DANFSe exports = %d, want 1 (no marks after a failure)", len(pending))
	}
}

func TestExportDANFSeZIP_NoRenderer(t *testing.T) {
	env := newNFeTestEnv(t)
	seedNFSe(t, env, "simple-prestada")
	out := filepath.Join(t.TempDir(), "danfses.zip")

	_, err := env.app.Exports.ExportDANFSeZIP(context.Background(), ExportInput{CNPJ: env.company.CNPJ, OutPath: out})
	if err == nil || !strings.Contains(err.Error(), "DANFSe não configurado") {
		t.Fatalf("err = %v, want DANFSe não configurado", err)
	}
	if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("output exists without a renderer (stat err %v)", err)
	}
}

func TestExportDANFSeZIP_WritesOnePDFPerDocument(t *testing.T) {
	env := newNFeTestEnv(t)
	seedNFSe(t, env, "simple-prestada")
	seedNFSe(t, env, "simple-tomada")
	env.app.Exports.DANFSeRenderer = fakeDANFSeRenderer{}
	ctx := context.Background()
	out := filepath.Join(t.TempDir(), "danfses.zip")

	res, err := env.app.Exports.ExportDANFSeZIP(ctx, ExportInput{CNPJ: env.company.CNPJ, OutPath: out, Incremental: true})
	if err != nil {
		t.Fatalf("ExportDANFSeZIP: %v", err)
	}
	if res.ExportedCount != 2 || res.Format != nfse.ExportKindDANFSe {
		t.Errorf("result = %+v, want 2 documents of kind %s", res, nfse.ExportKindDANFSe)
	}

	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(body), "pdf:") {
			t.Errorf("%s = %q, want the rendered PDF", f.Name, body)
		}
	}
	slices.Sort(names)
	want := []string{
		"2023-10/prestada/" + nfseChavePrestada + ".pdf",
		"2023-11/tomada/" + nfseChaveTomada + ".pdf",
	}
	if !slices.Equal(names, want) {
		t.Errorf("entries = %v, want %v", names, want)
	}

	pending, err := env.app.Exports.DocumentRepo.ListPendingExportDocuments(ctx, env.company.ID, nfse.DocumentFilter{}, nfse.ExportKindDANFSe)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Errorf("pending DANFSe exports = %d after the export, want 0", len(pending))
	}
}

func TestExportDANFSe_WritesOnePDF(t *testing.T) {
	env := newNFeTestEnv(t)
	xml := seedNFSe(t, env, "simple-prestada")
	env.app.Exports.DANFSeRenderer = fakeDANFSeRenderer{}
	ctx := context.Background()
	dir := t.TempDir()
	out := filepath.Join(dir, "nota.pdf")

	if err := env.app.Exports.ExportDANFSe(ctx, ExportDANFSeInput{CNPJ: env.company.CNPJ, ChaveAcesso: nfseChavePrestada, OutPath: out}); err != nil {
		t.Fatalf("ExportDANFSe: %v", err)
	}
	got, err := os.ReadFile(out) // #nosec G304 -- test temp file.
	if err != nil {
		t.Fatal(err)
	}
	if want := "pdf:" + string(xml[:10]); string(got) != want {
		t.Errorf("PDF = %q, want %q", got, want)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("files in the output dir = %v, want only the PDF", entries)
	}
	pending, err := env.app.Exports.DocumentRepo.ListPendingExportDocuments(ctx, env.company.ID, nfse.DocumentFilter{}, nfse.ExportKindDANFSe)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Errorf("pending DANFSe exports = %d after the export, want 0", len(pending))
	}
}

// TestExportDANFSe_FailuresWriteNothing: every failure before the write
// leaves the output directory empty and the document still pending.
func TestExportDANFSe_FailuresWriteNothing(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, env *nfeTestEnv) ExportDANFSeInput
		wantErr string
	}{
		{
			name: "no renderer",
			setup: func(_ *testing.T, env *nfeTestEnv) ExportDANFSeInput {
				env.app.Exports.DANFSeRenderer = nil
				return ExportDANFSeInput{CNPJ: env.company.CNPJ, ChaveAcesso: nfseChavePrestada}
			},
			wantErr: "DANFSe não configurado",
		},
		{
			name: "render failure",
			setup: func(_ *testing.T, env *nfeTestEnv) ExportDANFSeInput {
				env.app.Exports.DANFSeRenderer = fakeDANFSeRenderer{err: errors.New("layout quebrado")}
				return ExportDANFSeInput{CNPJ: env.company.CNPJ, ChaveAcesso: nfseChavePrestada}
			},
			wantErr: "renderizar DANFSe: layout quebrado",
		},
		{
			name: "stored XML missing",
			setup: func(t *testing.T, env *nfeTestEnv) ExportDANFSeInput {
				env.app.Exports.XMLStore = files.NewBlobStore(t.TempDir())
				return ExportDANFSeInput{CNPJ: env.company.CNPJ, ChaveAcesso: nfseChavePrestada}
			},
			wantErr: "ler XML original da chave " + nfseChavePrestada,
		},
		{
			name: "document without raw hash",
			setup: func(t *testing.T, env *nfeTestEnv) ExportDANFSeInput {
				if _, err := env.db.ExecContext(context.Background(), `UPDATE documents SET raw_hash = '' WHERE chave_acesso = ?`, nfseChavePrestada); err != nil {
					t.Fatal(err)
				}
				return ExportDANFSeInput{CNPJ: env.company.CNPJ, ChaveAcesso: nfseChavePrestada}
			},
			wantErr: "XML original não encontrado para a chave " + nfseChavePrestada,
		},
		{
			name: "document not found",
			setup: func(_ *testing.T, env *nfeTestEnv) ExportDANFSeInput {
				return ExportDANFSeInput{CNPJ: env.company.CNPJ, ChaveAcesso: nfseChaveTomada}
			},
			wantErr: "localizar documento",
		},
		{
			name: "document of another company",
			setup: func(t *testing.T, env *nfeTestEnv) ExportDANFSeInput {
				ctx := context.Background()
				cred := &credential.Credential{ID: "cred-other", Label: "Outra", CertPath: env.company.CredentialCertPath}
				if err := credential.NewStore(env.db).CreateCredential(ctx, cred); err != nil {
					t.Fatal(err)
				}
				other := storetest.TestCompany("comp-other", "11222333000181", dfe.EnvironmentProduction, cred)
				if err := company.NewStore(env.db).CreateCompany(ctx, other); err != nil {
					t.Fatal(err)
				}
				return ExportDANFSeInput{CNPJ: other.CNPJ, ChaveAcesso: nfseChavePrestada}
			},
			wantErr: "localizar documento",
		},
		{
			name: "unknown company",
			setup: func(_ *testing.T, _ *nfeTestEnv) ExportDANFSeInput {
				return ExportDANFSeInput{CNPJ: "11222333000181", ChaveAcesso: nfseChavePrestada}
			},
			wantErr: "empresa não encontrada",
		},
		{
			name: "empty chave",
			setup: func(_ *testing.T, env *nfeTestEnv) ExportDANFSeInput {
				return ExportDANFSeInput{CNPJ: env.company.CNPJ, ChaveAcesso: "  "}
			},
			wantErr: "chave de acesso não especificada",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newNFeTestEnv(t)
			seedNFSe(t, env, "simple-prestada")
			env.app.Exports.DANFSeRenderer = fakeDANFSeRenderer{}
			input := tc.setup(t, env)
			dir := t.TempDir()
			input.OutPath = filepath.Join(dir, "nota.pdf")
			ctx := context.Background()

			err := env.app.Exports.ExportDANFSe(ctx, input)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want %q", err, tc.wantErr)
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Errorf("files left after a failed export: %v", entries)
			}
			pending, err := env.app.Exports.DocumentRepo.ListPendingExportDocuments(ctx, env.company.ID, nfse.DocumentFilter{}, nfse.ExportKindDANFSe)
			if err != nil {
				t.Fatal(err)
			}
			if len(pending) != 1 {
				t.Errorf("pending DANFSe exports = %d, want 1 (no marks after a failure)", len(pending))
			}
		})
	}
}

func TestExportDANFSe_RequiresOutPath(t *testing.T) {
	env := newNFeTestEnv(t)
	err := env.app.Exports.ExportDANFSe(context.Background(), ExportDANFSeInput{CNPJ: env.company.CNPJ, ChaveAcesso: nfseChavePrestada})
	if err == nil || !strings.Contains(err.Error(), "caminho de saída não especificado") {
		t.Fatalf("err = %v, want the missing output path error", err)
	}
}

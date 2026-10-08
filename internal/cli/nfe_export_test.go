package cli

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vasfvitor/nanci/internal/dfe"
)

// nfeZipEntryNames returns the sorted entry names of the ZIP at path.
func nfeZipEntryNames(t *testing.T, path string) []string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	slices.Sort(names)
	return names
}

func TestNFeExportZip_Layout(t *testing.T) {
	env := newNFeTestRoot(t)
	env.seed("procnfe.xml", 1)
	env.seed("resnfe-cancelada.xml", 2)
	env.seed("proceventonfe-cancelamento.xml", 3)
	outPath := filepath.Join(t.TempDir(), "nfe.zip")

	if err := env.run("nfe", "export", "zip", "-c", nfeTestCNPJ, "--out", outPath); err != nil {
		t.Fatalf("export zip: %v", err)
	}
	got := env.out.String()
	for _, want := range []string{
		"Arquivo xml gerado com sucesso: " + outPath,
		"Documentos exportados: 1",
		"Resumos não exportados: 1 (use --incluir-resumos para incluí-los)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("export zip output lacks %q:\n%s", want, got)
		}
	}
	want := []string{"2026-09/destinatario/" + nfeChaveProc + "-procNFe.xml"}
	if names := nfeZipEntryNames(t, outPath); !slices.Equal(names, want) {
		t.Errorf("zip entries = %v, want %v", names, want)
	}

	if err := env.run("nfe", "export", "zip", "-c", nfeTestCNPJ, "--out", outPath, "--incremental"); err != nil {
		t.Fatalf("export zip --incremental: %v", err)
	}
	if got := env.out.String(); !strings.Contains(got, "Nenhum documento pendente para exportação incremental.") {
		t.Errorf("export zip --incremental output:\n%s", got)
	}
}

func TestNFeExportZip_IncluirResumos(t *testing.T) {
	env := newNFeTestRoot(t)
	env.seed("procnfe.xml", 1)
	env.seed("resnfe-cancelada.xml", 2)
	env.seed("proceventonfe-cancelamento.xml", 3)
	outPath := filepath.Join(t.TempDir(), "nfe.zip")

	if err := env.run("nfe", "export", "zip", "-c", nfeTestCNPJ, "--out", outPath, "--incluir-resumos"); err != nil {
		t.Fatalf("export zip --incluir-resumos: %v", err)
	}
	got := env.out.String()
	if !strings.Contains(got, "Documentos exportados: 2") || strings.Contains(got, "Resumos não exportados") {
		t.Errorf("export zip --incluir-resumos output:\n%s", got)
	}
	want := []string{
		"2026-08/destinatario/" + nfeChaveCancelada + "-resNFe.xml",
		"2026-08/destinatario/eventos/" + nfeChaveCancelada + "-110111-1.xml",
		"2026-09/destinatario/" + nfeChaveProc + "-procNFe.xml",
	}
	if names := nfeZipEntryNames(t, outPath); !slices.Equal(names, want) {
		t.Errorf("zip entries = %v, want %v", names, want)
	}
}

func TestNFeExportZip_ValidatesChave(t *testing.T) {
	env := newNFeTestRoot(t)
	env.seed("procnfe.xml", 1)
	outPath := filepath.Join(t.TempDir(), "nfe.zip")

	err := env.run("nfe", "export", "zip", "-c", nfeTestCNPJ, "--out", outPath, "--chave", "123")
	if !errors.Is(err, dfe.ErrInvalidAccessKey) {
		t.Fatalf("export zip --chave 123 = %v, want dfe.ErrInvalidAccessKey", err)
	}

	env = newNFeTestRoot(t) // flag values stick to a command tree
	env.seed("procnfe.xml", 1)
	if err := env.run("nfe", "export", "zip", "-c", nfeTestCNPJ, "--out", outPath, "--chave", " "+nfeChaveProc+" "); err != nil {
		t.Fatalf("export zip with a spaced --chave: %v", err)
	}
	if got := env.out.String(); !strings.Contains(got, "Documentos exportados: 1") {
		t.Errorf("export zip output:\n%s", got)
	}
}

func TestNFeExportXML_WritesStoredXML(t *testing.T) {
	env := newNFeTestRoot(t)
	env.seed("procnfe.xml", 1)
	outPath := filepath.Join(t.TempDir(), "nfe.xml")

	if err := env.run("nfe", "export", "xml", "-c", nfeTestCNPJ, "--chave", nfeChaveProc, "--out", outPath); err != nil {
		t.Fatalf("export xml: %v", err)
	}
	if got := env.out.String(); !strings.Contains(got, "XML exportado com sucesso: "+outPath) {
		t.Errorf("export xml output:\n%s", got)
	}
	got, err := os.ReadFile(outPath) // #nosec G304 -- test temp path.
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile(filepath.Join("..", "nfe", "testdata", "procnfe.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, fixture) {
		t.Error("exported XML differs from the stored fixture")
	}

	if err := env.run("nfe", "export", "xml", "-c", nfeTestCNPJ, "--chave", "123"); !errors.Is(err, dfe.ErrInvalidAccessKey) {
		t.Errorf("export xml --chave 123 = %v, want dfe.ErrInvalidAccessKey", err)
	}
}

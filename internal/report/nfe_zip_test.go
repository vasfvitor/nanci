package report

import (
	"archive/zip"
	"errors"
	"io"
	"path/filepath"
	"slices"
	"testing"

	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfe"
)

func TestNFeZipEntriesLayout(t *testing.T) {
	const proc, resumo = "35260911222333000181550010000012341123456787", "35260911222333000181550010000012351234567894"
	docs := []nfe.CompanyDocument{
		{
			Document:    nfe.Document{ChaveAcesso: proc, Competence: "2026-09", Completeness: nfe.CompletenessCompleta, RawHash: "hash-proc"},
			CompanyRole: nfe.CompanyRoleDestinatario,
		},
		{
			Document:    nfe.Document{ChaveAcesso: resumo, Completeness: nfe.CompletenessResumo, RawHash: "hash-res"},
			CompanyRole: nfe.CompanyRoleNone,
		},
	}
	events := map[string][]nfe.Event{
		proc: {
			{TpEvento: "210210", NSeqEvento: 1, Completeness: nfe.CompletenessCompleta, RawHash: "hash-ciencia"},
			{TpEvento: "110110", NSeqEvento: 2, Completeness: nfe.CompletenessResumo, RawHash: "hash-resevento"},
			{TpEvento: "110111", NSeqEvento: 1, Completeness: nfe.CompletenessCompleta},
		},
	}

	got := NFeZipEntries(docs, events)
	want := []ZipEntry{
		{Path: "2026-09/destinatario/" + proc + "-procNFe.xml", RawHash: "hash-proc"},
		{Path: "2026-09/destinatario/eventos/" + proc + "-210210-1.xml", RawHash: "hash-ciencia"},
		{Path: "sem-papel-fiscal/" + resumo + "-resNFe.xml", RawHash: "hash-res"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("entries =\n%+v\nwant\n%+v", got, want)
	}
}

func TestGenerateZIPWritesNFeEntries(t *testing.T) {
	baseDir := t.TempDir()
	store := files.NewBlobStore(baseDir)
	if err := store.Store("hash-proc", []byte("<nfeProc/>")); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(baseDir, "nfe.zip")
	entries := []ZipEntry{{Path: "2026-09/destinatario/chave-procNFe.xml", RawHash: "hash-proc"}}

	if err := GenerateZIP(entries, store, outPath); err != nil {
		t.Fatalf("GenerateZIP: %v", err)
	}
	archive, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = archive.Close() }()
	if len(archive.File) != 1 || archive.File[0].Name != entries[0].Path {
		t.Fatalf("archive files = %v", archive.File)
	}
	rc, err := archive.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rc.Close() }()
	if data, _ := io.ReadAll(rc); string(data) != "<nfeProc/>" {
		t.Errorf("entry content = %q", data)
	}

	missing := []ZipEntry{{Path: "x/chave-procNFe.xml", RawHash: "missing"}}
	if err := GenerateZIP(missing, store, filepath.Join(baseDir, "missing.zip")); !errors.Is(err, files.ErrBlobNotFound) {
		t.Errorf("missing blob err = %v, want ErrBlobNotFound", err)
	}
}

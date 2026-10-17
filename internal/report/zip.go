package report

import (
	"archive/zip"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/vasfvitor/nanci/internal/cte"
	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfe"
	"github.com/vasfvitor/nanci/internal/nfse"
)

// ZipEntry is one file of an archive built from the stored blob RawHash and
// written at Path: the XML itself, or the PDF rendered from it.
type ZipEntry struct {
	Path    string
	RawHash string
	// Cancelada asks the PDF renderer for the cancellation watermark.
	Cancelada bool
}

// NFSeZipEntries lays out the NFS-e archive: each document goes to
// <competencia>/<papel>/<chave>.xml. Documents without a stored XML are left
// out.
func NFSeZipEntries(rows []NFSeRow) []ZipEntry {
	var entries []ZipEntry
	for _, row := range rows {
		if row.RawHash == "" {
			continue
		}
		entryPath := path.Join(RoleFolder(row.Competence, string(row.CompanyRole)), row.ChaveAcesso+".xml")
		entries = append(entries, ZipEntry{Path: entryPath, RawHash: row.RawHash})
	}
	return entries
}

// NFSePDFEntries lays out the DANFSe archive: each document goes to
// <competencia>/<papel>/<chave>.pdf. Documents without a stored XML are left
// out.
func NFSePDFEntries(docs []nfse.CompanyDocument) []ZipEntry {
	var entries []ZipEntry
	for _, doc := range docs {
		if doc.RawHash == "" {
			continue
		}
		entries = append(entries, ZipEntry{
			Path:      path.Join(RoleFolder(doc.Competence, string(doc.CompanyRole)), string(doc.ChaveAcesso)+".pdf"),
			RawHash:   doc.RawHash,
			Cancelada: doc.Status == nfse.DocumentStatusCancelada,
		})
	}
	return entries
}

// NFePDFEntries lays out the DANFE archive: each document goes to
// <competencia>/<papel>/<chave>.pdf. A resumo has no DANFE and documents
// without a stored XML are left out.
func NFePDFEntries(docs []nfe.CompanyDocument) []ZipEntry {
	var entries []ZipEntry
	for _, doc := range docs {
		if doc.Completeness == nfe.CompletenessResumo || doc.RawHash == "" {
			continue
		}
		entries = append(entries, ZipEntry{
			Path:      path.Join(RoleFolder(doc.Competence, string(doc.CompanyRole)), string(doc.ChaveAcesso)+".pdf"),
			RawHash:   doc.RawHash,
			Cancelada: doc.Situacao == nfe.SituacaoCancelada,
		})
	}
	return entries
}

// CTePDFEntries lays out the DACTE archive: each document goes to
// <competencia>/<papel>/<chave>.pdf. Kinds the renderer has no layout for
// and documents without a stored XML are left out.
func CTePDFEntries(docs []cte.CompanyDocument) []ZipEntry {
	var entries []ZipEntry
	for _, doc := range docs {
		if !doc.HasDACTE() || doc.RawHash == "" {
			continue
		}
		entries = append(entries, ZipEntry{
			Path:      path.Join(RoleFolder(doc.Competence, string(doc.CompanyRole)), string(doc.ChaveAcesso)+".pdf"),
			RawHash:   doc.RawHash,
			Cancelada: doc.Situacao == cte.SituacaoCancelada,
		})
	}
	return entries
}

// eventZipEntry places an NF-e or CT-e event of chave in the eventos folder
// of its document: <folder>/eventos/<chave>-<tpEvento>-<nSeq>.xml.
func eventZipEntry(folder, chave, tpEvento string, nSeq int, rawHash string) ZipEntry {
	name := chave + "-" + tpEvento + "-" + strconv.Itoa(nSeq) + ".xml"
	return ZipEntry{Path: path.Join(folder, "eventos", name), RawHash: rawHash}
}

// RoleFolder is the archive folder of a document: <competencia>/<papel>, or
// just <papel> when the competência is unknown. A document without a fiscal
// role goes to "sem-papel-fiscal".
func RoleFolder(competence, role string) string {
	if role == "" || role == "none" {
		role = "sem-papel-fiscal"
	}
	if competence == "" {
		return role
	}
	return path.Join(competence, role)
}

// GenerateZIP writes the stored XML of each entry into a new ZIP archive at
// outPath.
func GenerateZIP(entries []ZipEntry, xmlStore files.XMLStore, outPath string) error {
	return writeZIP(outPath, entries, func(entry ZipEntry) ([]byte, error) {
		data, err := xmlStore.Get(entry.RawHash)
		if err != nil {
			return nil, fmt.Errorf("arquivo físico XML não encontrado para %s: %w", entry.Path, err)
		}
		return data, nil
	})
}

// GeneratePDFZIP renders the stored XML of each entry with render and writes
// the PDFs into a new ZIP archive at outPath.
func GeneratePDFZIP(entries []ZipEntry, xmlStore files.XMLStore, render func(xml []byte, cancelada bool) ([]byte, error), outPath string) error {
	return writeZIP(outPath, entries, func(entry ZipEntry) ([]byte, error) {
		data, err := xmlStore.Get(entry.RawHash)
		if err != nil {
			return nil, fmt.Errorf("arquivo físico XML não encontrado para %s: %w", entry.Path, err)
		}
		pdf, err := render(data, entry.Cancelada)
		if err != nil {
			return nil, fmt.Errorf("gerar PDF %s: %w", entry.Path, err)
		}
		return pdf, nil
	})
}

// writeZIP creates the archive at outPath and writes content(entry) at each
// entry's path, in order.
func writeZIP(outPath string, entries []ZipEntry, content func(ZipEntry) ([]byte, error)) (err error) {
	zipFile, err := os.Create(outPath) // #nosec G304 -- destination is explicitly selected by the local user.
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer func() {
		if cerr := zipFile.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close zip file: %w", cerr)
		}
	}()

	zipWriter := zip.NewWriter(zipFile)
	defer func() {
		if cerr := zipWriter.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close zip writer: %w", cerr)
		}
	}()

	for _, entry := range entries {
		data, err := content(entry)
		if err != nil {
			return err
		}
		writer, err := zipWriter.Create(entry.Path)
		if err != nil {
			return fmt.Errorf("failed to create zip entry %s: %w", entry.Path, err)
		}
		if _, err := writer.Write(data); err != nil {
			return fmt.Errorf("failed to write zip entry %s: %w", entry.Path, err)
		}
	}
	return nil
}

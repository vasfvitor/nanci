package report

import (
	"archive/zip"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/vasfvitor/nanci/internal/files"
)

// ZipEntry is one file of an XML archive: the stored blob RawHash written
// at Path.
type ZipEntry struct {
	Path    string
	RawHash string
}

// NFSeZipEntries lays out the NFS-e archive: each document goes to
// <competencia>/<papel>/<chave>.xml. Documents without a stored XML are left
// out.
func NFSeZipEntries(rows []ReportRow) []ZipEntry {
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

// GenerateZIP writes the entries into a new ZIP archive at outPath.
func GenerateZIP(entries []ZipEntry, xmlStore files.XMLStore, outPath string) (err error) {
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
		data, err := xmlStore.Get(entry.RawHash)
		if err != nil {
			return fmt.Errorf("arquivo físico XML não encontrado para %s: %w", entry.Path, err)
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

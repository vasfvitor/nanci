package report

import (
	"archive/zip"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/vasfvitor/nanci/internal/files"
	"github.com/vasfvitor/nanci/internal/nfe"
)

// NFeZipEntry is one file of the NF-e XML archive: the stored blob RawHash
// written at Path.
type NFeZipEntry struct {
	Path    string
	RawHash string
}

// BuildNFeZipEntries lays out the archive: each document goes to
// <competencia>/<papel>/<chave>-procNFe.xml (or -resNFe.xml for a resumo)
// and each complete event of it to
// <competencia>/<papel>/eventos/<chave>-<tpEvento>-<nSeq>.xml.
// eventsByChave holds the events of each document's chave.
func BuildNFeZipEntries(docs []nfe.CompanyDocument, eventsByChave map[string][]nfe.Event) []NFeZipEntry {
	var entries []NFeZipEntry
	for _, doc := range docs {
		folder := nfeFolder(doc)
		chave := string(doc.ChaveAcesso)

		suffix := "-procNFe.xml"
		if doc.Completeness == nfe.CompletenessResumo {
			suffix = "-resNFe.xml"
		}
		if doc.RawHash != "" {
			entries = append(entries, NFeZipEntry{Path: path.Join(folder, chave+suffix), RawHash: doc.RawHash})
		}

		for _, ev := range eventsByChave[chave] {
			if ev.Completeness != nfe.CompletenessCompleta || ev.RawHash == "" {
				continue
			}
			name := chave + "-" + ev.TpEvento + "-" + strconv.Itoa(ev.NSeqEvento) + ".xml"
			entries = append(entries, NFeZipEntry{Path: path.Join(folder, "eventos", name), RawHash: ev.RawHash})
		}
	}
	return entries
}

func nfeFolder(doc nfe.CompanyDocument) string {
	role := string(doc.CompanyRole)
	if role == "" || doc.CompanyRole == nfe.CompanyRoleNone {
		role = "sem-papel-fiscal"
	}
	if doc.Competence == "" {
		return role
	}
	return path.Join(doc.Competence, role)
}

// GenerateNFeZIP writes the entries into a new ZIP archive at outPath.
func GenerateNFeZIP(entries []NFeZipEntry, xmlStore files.XMLStore, outPath string) (err error) {
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
			return fmt.Errorf("arquivo XML não encontrado para %s: %w", entry.Path, err)
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

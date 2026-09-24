package report

import (
	"path"

	"github.com/vasfvitor/nanci/internal/nfe"
)

// NFeZipEntries lays out the archive: each document goes to
// <competencia>/<papel>/<chave>-procNFe.xml (or -resNFe.xml for a resumo)
// and each complete event of it to
// <competencia>/<papel>/eventos/<chave>-<tpEvento>-<nSeq>.xml.
// eventsByChave holds the events of each document's chave.
func NFeZipEntries(docs []nfe.CompanyDocument, eventsByChave map[string][]nfe.Event) []ZipEntry {
	var entries []ZipEntry
	for _, doc := range docs {
		folder := RoleFolder(doc.Competence, string(doc.CompanyRole))
		chave := string(doc.ChaveAcesso)

		suffix := "-procNFe.xml"
		if doc.Completeness == nfe.CompletenessResumo {
			suffix = "-resNFe.xml"
		}
		if doc.RawHash != "" {
			entries = append(entries, ZipEntry{Path: path.Join(folder, chave+suffix), RawHash: doc.RawHash})
		}

		for _, ev := range eventsByChave[chave] {
			if ev.Completeness != nfe.CompletenessCompleta || ev.RawHash == "" {
				continue
			}
			entries = append(entries, eventZipEntry(folder, chave, ev.TpEvento, ev.NSeqEvento, ev.RawHash))
		}
	}
	return entries
}

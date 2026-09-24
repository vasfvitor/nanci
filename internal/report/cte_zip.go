package report

import (
	"path"

	"github.com/vasfvitor/nanci/internal/cte"
)

// CTeZipEntries lays out the archive: each document goes to
// <competencia>/<papel>/<chave>-procCTe.xml (-procCTeOS, -procGTVe or
// -procCTeSimp by kind) and each of its events to
// <competencia>/<papel>/eventos/<chave>-<tpEvento>-<nSeq>.xml.
// eventsByChave holds the events of each document's chave.
func CTeZipEntries(docs []cte.CompanyDocument, eventsByChave map[string][]cte.Event) []ZipEntry {
	var entries []ZipEntry
	for _, doc := range docs {
		folder := RoleFolder(doc.Competence, string(doc.CompanyRole))
		chave := string(doc.ChaveAcesso)

		if doc.RawHash != "" {
			entries = append(entries, ZipEntry{Path: path.Join(folder, chave+cteFileSuffix(doc.TipoDocumento)), RawHash: doc.RawHash})
		}

		for _, ev := range eventsByChave[chave] {
			if ev.RawHash == "" {
				continue
			}
			entries = append(entries, eventZipEntry(folder, chave, ev.TpEvento, ev.NSeqEvento, ev.RawHash))
		}
	}
	return entries
}

// cteFileSuffix is the file name suffix of a stored transport document, after
// its chave: "-procCTe.xml", "-procCTeOS.xml", "-procGTVe.xml" or
// "-procCTeSimp.xml".
func cteFileSuffix(tipo cte.TipoDocumento) string {
	switch tipo {
	case cte.TipoDocumentoCTeOS:
		return "-procCTeOS.xml"
	case cte.TipoDocumentoGTVe:
		return "-procGTVe.xml"
	case cte.TipoDocumentoCTeSimplificado:
		return "-procCTeSimp.xml"
	default:
		return "-procCTe.xml"
	}
}

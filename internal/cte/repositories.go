package cte

import (
	"errors"
	"time"

	"github.com/vasfvitor/nanci/internal/dfe"
)

// Export kinds tracked in the CT-e export marks. Each kind keeps its own
// marks, so exporting the XML does not mark the spreadsheet as exported.
const (
	ExportKindXML   = "xml"
	ExportKindCSV   = "csv"
	ExportKindXLSX  = "xlsx"
	ExportKindDACTE = "dacte"
)

// ErrDocumentNotFound is returned when the company does not see a chave.
var ErrDocumentNotFound = errors.New("CT-e não encontrado para a empresa")

// DocumentFilter selects company CT-e rows. Zero values do not filter.
type DocumentFilter struct {
	Competence string // "YYYY-MM"
	Situacao   Situacao
	// Role matches the primary role or any of the company's Papeis.
	Role         CompanyRole
	Modelo       string // "57", "64" or "67"
	EmitenteCNPJ string
	TomadorCNPJ  string
	// NFeChave keeps the CT-e that transported this NF-e.
	NFeChave     string
	ChavesAcesso []string
	TpAmb        string // "1" produção, "2" homologação
	// OnlyUnread keeps rows the company has not marked as viewed.
	OnlyUnread bool
	Limit      int
}

// Counts summarizes one company's CT-e by primary role.
type Counts struct {
	ByRole map[CompanyRole]int
}

// ResetCounts is what a company CT-e reset removes, or would remove.
type ResetCounts struct {
	CompanyDocuments int // the company's rows in company_cte_documents
	Documents        int // cte_documents no other company sees
	Events           int // cte_events of those documents
	ExportMarks      int
}

// Statuses of an outbound prestação em desacordo, stored in
// cte_desacordos.status. They match the NF-e manifestação ones.
const (
	DesacordoStatusRegistrada   = "registrada"
	DesacordoStatusJaRegistrada = "ja_registrada"
	DesacordoStatusRejeitada    = "rejeitada"
	DesacordoStatusErro         = "erro" // the request got no SEFAZ answer
)

// DesacordoRecord is one prestação em desacordo sent to SEFAZ, whatever
// the answer. Fields are plain values so the store does not depend on the
// SEFAZ client.
type DesacordoRecord struct {
	CompanyID   dfe.CompanyID
	ChaveAcesso string
	TpAmb       string // tpAmb the event was sent to: 1 produção, 2 homologação
	NSeqEvento  int
	Observacao  string
	EventAt     time.Time // dhEvento sent
	Status      string    // one of the DesacordoStatus* constants
	CStat       string
	XMotivo     string
	Protocolo   string
	// RegisteredAt is dhRegEvento, when SEFAZ registered the event.
	RegisteredAt *time.Time

	RequestRawHash  string
	ResponseRawHash string
	// Event is the registered event, read from its procEventoCTe, to store
	// in cte_events; nil unless SEFAZ registered it now.
	Event *Event
}

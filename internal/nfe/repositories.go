package nfe

import (
	"errors"
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// ExportKindXML is the only export kind tracked for NF-e.
const ExportKindXML = "xml"

// ErrDocumentNotFound is returned when the company does not see a chave.
var ErrDocumentNotFound = errors.New("NF-e not found for the company")

// DocumentFilter selects company NF-e rows. Zero values do not filter.
type DocumentFilter struct {
	Competence   string // "YYYY-MM"
	Situacao     Situacao
	Completeness Completeness
	Role         CompanyRole
	Manifestacao Manifestacao
	EmitenteCNPJ string
	ChavesAcesso []string
	OnlyUnread   bool
	// PendingManifestation keeps authorized documents where the company is
	// the destinatário and has no conclusive manifestação yet.
	PendingManifestation bool
	Limit                int
}

// Counts summarizes one company's NF-e.
type Counts struct {
	ByRole    map[CompanyRole]int
	Resumos   int
	Completas int
}

// ResetCounts is what a company NF-e reset removes, or would remove, and the
// manifestações it keeps.
type ResetCounts struct {
	CompanyDocuments   int // the company's rows in company_nfe_documents
	Documents          int // nfe_documents no other company sees
	Events             int // nfe_events of those documents, and the company's own events without a document
	ExportMarks        int
	ManifestationsKept int // nfe_manifestations stay as the audit trail of what was sent
}

// Statuses of an outbound manifestação, stored in nfe_manifestations.status.
const (
	ManifestationStatusRegistrada   = "registrada"
	ManifestationStatusJaRegistrada = "ja_registrada"
	ManifestationStatusRejeitada    = "rejeitada"
	ManifestationStatusErro         = "erro" // the lote got no SEFAZ answer
)

// ManifestationRecord is the outcome of one event of a lote sent to SEFAZ.
// Fields are plain values so the store does not depend on the SEFAZ client.
type ManifestationRecord struct {
	CompanyID     nfse.CompanyID
	CompanyCNPJ   string
	IDLote        string
	TpAmb         string // tpAmb the lote was sent to: 1 produção, 2 homologação
	ChaveAcesso   string
	TpEvento      string
	NSeqEvento    int
	EventAt       *time.Time // dhEvento sent
	Description   string     // descEvento sent
	Justificativa string
	Status        string // one of the ManifestationStatus* constants
	CStat         string
	XMotivo       string
	Protocolo     string
	RegisteredAt  *time.Time

	RequestRawHash    string
	ResponseRawHash   string
	ProcEventoRawHash string
}

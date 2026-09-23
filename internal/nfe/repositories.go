package nfe

import (
	"time"

	"github.com/vasfvitor/nanci/internal/nfse"
)

// ExportKindXML is the only export kind tracked for NF-e.
const ExportKindXML = "xml"

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
	// IssueDateGTE keeps documents issued on or after this day.
	IssueDateGTE *time.Time
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
	// PendingCiencia counts authorized documents addressed to the company
	// without any manifestação.
	PendingCiencia int
	// PendingConclusiva counts authorized documents addressed to the company
	// with ciência but no conclusive manifestação.
	PendingConclusiva int
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

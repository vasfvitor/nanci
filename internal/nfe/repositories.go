package nfe

import "time"

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
